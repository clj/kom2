package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"runtime/cgo"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/exp/constraints"
	"golang.org/x/sync/errgroup"
)

// #if defined(_WIN32)
//   #include <windows.h>
// #endif
// #include <stdint.h>
// #include <stdlib.h>
// #include <string.h>
// #include <sqltypes.h>
// #include <sql.h>
// #include <sqlext.h>
import "C"

func Convert(value any, t string) (any, error) {
	switch t {
	case "int":
	case "float":
	case "string":
	case "":
		return value, nil
	default:
		return nil, fmt.Errorf("invalid destination type %s", t)

	}

	switch v := value.(type) {
	case int64:
		switch t {
		case "int":
			return v, nil
		case "float":
			return float64(v), nil
		case "string":
			return strconv.FormatInt(v, 10), nil
		}
	case float64:
		switch t {
		case "int":
			return int64(v), nil
		case "float":
			return v, nil
		case "string":
			return strconv.FormatFloat(v, 'G', -1, 64), nil
		}
	case string:
		switch t {
		case "int":
			return strconv.ParseInt(v, 0, 64)
		case "float":
			return strconv.ParseFloat(v, 64)
		case "string":
			return v, nil
		}
	case bool:
		switch t {
		case "int":
			if v {
				return 1, nil
			}
			return 0, nil
		case "float":
			if v {
				return 1.0, nil
			}
			return 0.0, nil
		case "string":
			if v {
				return "1", nil
			}
			return "0", nil
		}
	}

	return nil, fmt.Errorf("unknown source type %T for value %v", value, value)
}

func toGoString[T C.int | C.short | C.long](str *C.SQLCHAR, len T) string {
	if str == nil {
		return ""
	}
	if len == C.SQL_NTS {
		return C.GoString((*C.char)(unsafe.Pointer(str)))
	}
	return C.GoStringN((*C.char)(unsafe.Pointer(str)), C.int(len))
}

func (connHandle *connectionHandle) initConnection(dsn, connectionString, userName, password string) C.SQLRETURN {
	args := make(map[string]string)
	for _, arg := range strings.Split(connectionString, ";") {
		sarg := strings.SplitN(arg, "=", 2)
		if len(sarg) == 1 {
			args[strings.ToLower(sarg[0])] = ""
		} else {
			args[strings.ToLower(sarg[0])] = sarg[1]
		}
	}
	conStrArg := func(arg, defaultValue string) string {
		if value, ok := args[arg]; ok {
			return value
		}
		return defaultValue
	}

	dsn = conStrArg("dsn", dsn)

	var fetchParametersStr, fetchMetadataStr string

	// Config file is lowest priority
	if dsn != "" {
		connHandle.inventreeConfig.server = SQLGetPrivateProfileString(dsn, "server", "", ".odbc.ini")
		connHandle.inventreeConfig.userName = SQLGetPrivateProfileString(dsn, "username", "", ".odbc.ini")
		connHandle.inventreeConfig.password = SQLGetPrivateProfileString(dsn, "password", "", ".odbc.ini")
		connHandle.inventreeConfig.apiToken = SQLGetPrivateProfileString(dsn, "apitoken", "", ".odbc.ini")
		fetchParametersStr = SQLGetPrivateProfileString(dsn, "apitoken", "", ".odbc.ini")
		fetchMetadataStr = SQLGetPrivateProfileString(dsn, "apitoken", "", ".odbc.ini")

	}

	// Then connection string is higher
	connHandle.inventreeConfig.server = conStrArg("server", connHandle.inventreeConfig.server)
	connHandle.inventreeConfig.userName = conStrArg("username", connHandle.inventreeConfig.userName)
	connHandle.inventreeConfig.password = conStrArg("password", connHandle.inventreeConfig.password)
	connHandle.inventreeConfig.apiToken = conStrArg("apitoken", connHandle.inventreeConfig.apiToken)
	fetchParametersStr = conStrArg("fetchparameters", fetchParametersStr)
	fetchMetadataStr = conStrArg("fetchparameters", fetchMetadataStr)

	// Then explicit username and password is highest
	if userName != "" {
		connHandle.inventreeConfig.userName = userName
	}
	if password != "" {
		connHandle.inventreeConfig.password = password
	}

	switch strings.ToLower(fetchParametersStr) {
	case "yes":
		connHandle.inventreeConfig.fetchParameters = true
	case "no":
		connHandle.inventreeConfig.fetchParameters = false
	case "":
		connHandle.inventreeConfig.fetchParameters = true
	default:
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "fetchParameters accepts 'yes' or 'no"})
	}

	switch strings.ToLower(fetchMetadataStr) {
	case "yes":
		connHandle.inventreeConfig.fetchMetadata = true
	case "no":
		connHandle.inventreeConfig.fetchMetadata = false
	case "":
		connHandle.inventreeConfig.fetchMetadata = false
	default:
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "fetchMetadata accepts 'yes' or 'no"})
	}

	if connHandle.inventreeConfig.server == "" {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "No Server specified"})
	}
	connHandle.inventreeConfig.server = strings.TrimSuffix(connHandle.inventreeConfig.server, "/")
	if connHandle.inventreeConfig.apiToken == "" && (connHandle.inventreeConfig.userName == "" || connHandle.inventreeConfig.password == "") {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "No APIToken or Username+Password specified"})
	}

	if connHandle.inventreeConfig.apiToken == "" {
		var token string
		token, err := connHandle.getApiToken(connHandle.inventreeConfig.userName, connHandle.inventreeConfig.password) // why pass these?
		if err != nil {
			return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "Failed to fetch API Token", Err: err})
		}
		connHandle.inventreeConfig.apiToken = token
	}

	if err := connHandle.updateCategoryMapping(); err != nil {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "Error updating category list", Err: err})
	}

	return C.SQL_SUCCESS
}

//export SQLConnect
func SQLConnect(ConnectionHandle C.SQLHDBC,
	ServerName *C.SQLCHAR, NameLength1 C.SQLSMALLINT,
	UserName *C.SQLCHAR, NameLength2 C.SQLSMALLINT,
	Authentication *C.SQLCHAR, NameLength3 C.SQLSMALLINT,
) C.SQLRETURN {
	serverName := toGoString(ServerName, NameLength1)
	userName := toGoString(UserName, NameLength2)
	password := toGoString(Authentication, NameLength3)

	connHandle := cgo.Handle(ConnectionHandle).Value().(*connectionHandle)
	return connHandle.initConnection(serverName, "", userName, password)
}

//export SQLDriverConnect
func SQLDriverConnect(
	ConnectionHandle C.SQLHDBC,
	WindowHandle C.SQLHWND,
	InConnectionString *C.SQLCHAR,
	StringLength1 C.SQLSMALLINT,
	OutConnectionString *C.SQLCHAR,
	BufferLength C.SQLSMALLINT,
	StringLength2Ptr *C.SQLSMALLINT,
	DriverCompletion C.SQLUSMALLINT,
) C.SQLRETURN {
	inConnectionString := toGoString(InConnectionString, StringLength1)

	connHandle := cgo.Handle(ConnectionHandle).Value().(*connectionHandle)
	return connHandle.initConnection("", inConnectionString, "", "")
}

type DriverError struct {
	SqlState    string
	NativeError string
	Message     string
	Err         error
}

func (e *DriverError) Error() string { return e.SqlState + ": " + e.Message }
func (e *DriverError) Unwrap() error { return e.Err }
func (e *DriverError) SetAndReturnError(handle errorInfo) C.SQLRETURN {
	handle.errorInfo = e

	return C.SQL_ERROR
}

func SetAndReturnError(handle interface{}, err *DriverError) C.SQLRETURN {
	switch h := handle.(type) {
	case *connectionHandle:
		h.errorInfo.errorInfo = err
	case *statementHandle:
		h.errorInfo.errorInfo = err
	default:
		panic("FIXME!")
	}

	return C.SQL_ERROR
}

type errorInfo struct {
	errorInfo *DriverError
}

type environmentHandle struct {
	errorInfo

	httpClient *http.Client
}

func (e *environmentHandle) init() {
	e.httpClient = &http.Client{}
}

type connectionHandle struct {
	errorInfo

	env             *environmentHandle
	inventreeConfig struct {
		server          string
		userName        string
		password        string
		apiToken        string
		fetchParameters bool
		fetchMetadata   bool
	}
	categoryMapping map[string]int
	// This cache isn't ideal because it never expires. However, for KiCad
	// I don't think it matters much, since KiCad will refresh its full list
	// of parts before individual parts can be selected, which means this
	// cache should be up to date with what parts can actually be selected.
	ipnToPkMap map[string]any
}

func (c *connectionHandle) init(envHandle *environmentHandle) {
	c.env = envHandle
}

func (c *connectionHandle) updateIpnToPkMap(parts *[]map[string]any) {
	if c.ipnToPkMap == nil {
		c.ipnToPkMap = make(map[string]any)
	}

	for _, part := range *parts {
		if part["IPN"] == nil {
			continue
		}
		pk, err := part["pk"].(json.Number).Int64()
		if err != nil {
			panic("Was unable to convert 'pk' to an int64")
		}
		ipn := part["IPN"].(string)

		if ipn == "" {
			continue
		}
		c.ipnToPkMap[ipn] = pk
	}
}

func (c *connectionHandle) getApiToken(userName, password string) (string, error) {
	request, err := http.NewRequest("GET", c.inventreeConfig.server+"/api/user/token", nil)
	if err != nil {
		return "", err
	}
	request.SetBasicAuth(userName, password)
	response, err := c.env.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code %s", response.Status)
	}

	type Token struct {
		Token string `json:"token"`
	}
	decoder := json.NewDecoder(response.Body)
	val := &Token{}
	err = decoder.Decode(val)
	if err != nil {
		return "", err
	}
	return val.Token, nil
}

func (c *connectionHandle) apiGet(resource string, args map[string]string, result any) error {
	request, err := http.NewRequest("GET", c.inventreeConfig.server+resource, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Token %s", c.inventreeConfig.apiToken))
	if args != nil {
		q := request.URL.Query()
		for key, val := range args {
			q.Add(key, val)
		}
		request.URL.RawQuery = q.Encode()
	}
	response, err := c.env.httpClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code %s", response.Status)
	}

	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	err = decoder.Decode(result)
	if err != nil {
		return err
	}

	return nil
}

func (c *connectionHandle) updateCategoryMapping() error {
	type category struct {
		Pk         int    `json:"pk"`
		Pathstring string `json:"pathstring"`
	}
	categories := []category{}
	if err := c.apiGet("/api/part/category/", nil, &categories); err != nil {
		return err
	}

	c.categoryMapping = make(map[string]int)
	for _, category := range categories {
		c.categoryMapping[category.Pathstring] = category.Pk
	}
	return nil
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (s *statementHandle) fetchAllParts(category string, parts *[]map[string]any) error {
	args := make(map[string]string)
	categoryId, ok := s.conn.categoryMapping[category]
	if !ok {
		return &DriverError{SqlState: "HY000", Message: fmt.Sprintf("Category does not exist in InvenTree: %s", category)}
	}
	args["category"] = strconv.Itoa(categoryId)

	return s.conn.apiGet("/api/part/", args, parts)
}

func mangleParameters(params []map[string]any) map[string]any {
	result := make(map[string]any)
	for _, param := range params {
		name := param["template_detail"].(map[string]any)["name"].(string)
		data := param["data"].(string)

		result[name] = data
	}

	return result
}

func (s *statementHandle) fetchPart(category string, column string, value any, parts *[]map[string]any) error {
	var part map[string]any
	var partMetadata map[string]any
	var partParameters map[string]any

	var pkValue any
	switch column {
	case "pk":
		pkValue = value
	case "IPN":
		if s.conn.ipnToPkMap == nil {
			var tmpParts []map[string]any
			if err := s.fetchAllParts(category, &tmpParts); err != nil {
				return err
			}

			s.conn.updateIpnToPkMap(&tmpParts)
		}

		var ok bool
		if pkValue, ok = s.conn.ipnToPkMap[value.(string)]; !ok {
			return nil
		}

		// XXX: could optimise away the next fetch of parts, since we should already
		//      have had the the required part returned when fetching all parts.
		//      but this is not a path that should ever be hit when fetching from KiCad.
		//      This is mostly to make running manual queries not annoying.

	default:
		panic(fmt.Sprintf("invalid filter column: %s", column))
	}

	getPart := func() error {
		if err := s.conn.apiGet(fmt.Sprintf("/api/part/%v/", pkValue), nil, &part); err != nil {
			return err
		}
		return nil
	}

	g := new(errgroup.Group)

	g.Go(getPart)

	if s.conn.inventreeConfig.fetchMetadata {
		g.Go(func() error {
			if err := s.conn.apiGet(fmt.Sprintf("/api/part/%v/metadata/", pkValue), nil, &partMetadata); err != nil {
				return err
			}

			return nil
		})
	}

	if s.conn.inventreeConfig.fetchParameters {
		g.Go(func() error {
			var rawPartParameters []map[string]any
			args := make(map[string]string)
			value, _ := Convert(pkValue, "string") // maybe just sprintf?
			args["part"] = value.(string)

			if err := s.conn.apiGet("/api/part/parameter/", args, &rawPartParameters); err != nil {
				return err
			}

			partParameters = mangleParameters(rawPartParameters)

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	var flatten func(string, map[string]any)
	flatten = func(path string, data map[string]any) {
		for key, value := range data {
			if path == "" {
				key = key
			} else {
				key = path + "." + key
			}

			if value == nil {
				part[key] = value // ???
				continue
			}

			switch reflect.TypeOf(value).Kind().String() {
			case "array":
				continue
			case "map":
				flatten(key, value.(map[string]any))
			default:
				part[key] = value
			}
		}
	}

	if partMetadata != nil {
		flatten("", partMetadata)
	}
	for key, parameter := range partParameters {
		part["parameter."+key] = parameter
	}
	*parts = append(*parts, part)

	return nil
}

func (s *statementHandle) populateColDesc(data *[]map[string]any) {
	columns := make(map[string]*desc)

	for _, row := range *data {
		for key := range row {
			if _, ok := columns[key]; !ok {
				size := 0 // See: http://www.ch-werner.de/sqliteodbc/html/sqlite3odbc_8c.html#a107
				var dataType C.short
				switch value := row[key].(type) {
				case json.Number:
					if _, err := value.Int64(); err == nil {
						dataType = C.SQL_BIGINT
						size = 20
					} else if _, err := value.Float64(); err == nil {
						dataType = C.SQL_DOUBLE
						size = 54
					}
				case string:
					dataType = C.SQL_VARCHAR
					size = 255
				case float64:
					dataType = C.SQL_DOUBLE
					size = 54
				case bool:
					dataType = C.SQL_INTEGER
					size = 10
				}
				columns[key] = &desc{name: key, dataType: dataType, colSize: size, nullable: C.SQL_NULLABLE}
			}
		}
	}

	s.columnNames = keys(columns)
	sort.Strings(s.columnNames)

	for _, name := range s.columnNames {
		s.def = append(s.def, columns[name])
	}
}

func (s *statementHandle) populateData(data *[]map[string]any) {
	for _, row := range *data {
		rowData := make([]any, len(s.def))
		for idx, colName := range s.columnNames {
			if value, ok := row[colName]; ok {
				rowData[idx] = value
			}
		}
		s.data = append(s.data, rowData)
	}
}

type bind struct {
	TargetType       C.SQLSMALLINT
	TargetValuePtr   C.SQLPOINTER
	BufferLength     C.SQLLEN
	StrLen_or_IndPtr *C.SQLLEN
}

type param struct {
	ValueType         C.SQLSMALLINT
	ParameterValuePtr C.SQLPOINTER
	BufferLength      C.SQLLEN
}

type desc struct {
	name          string
	dataType      C.short
	colSize       int
	decimalDigits int
	nullable      int
}

type cond struct {
	column string
	value  string
}

type stmt struct {
	table     string
	condition *cond
}
type statementHandle struct {
	errorInfo

	conn           *connectionHandle
	columnNames    []string
	def            []*desc
	binds          []*bind
	params         []*param
	data           [][]any
	index          int
	rowsFetchedPtr *C.SQLULEN
	statement      *stmt
}

func (s *statementHandle) init(connHandle *connectionHandle) {
	s.conn = connHandle
	s.index = -1
}

func (s *statementHandle) populateBinds() {
	for idx, bind := range s.binds {
		if bind == nil {
			continue
		}

		value := s.data[s.index][idx]
		populateData(value, bind.TargetType, bind.TargetValuePtr, bind.BufferLength, bind.StrLen_or_IndPtr)
	}
}

func copyGoStringToCString(dst *C.uchar, src string, length int) {
	cSrc := C.CString(src + "\x00")
	defer C.free(unsafe.Pointer(cSrc))
	C.strncpy((*C.char)(unsafe.Pointer(dst)), cSrc, C.size_t(length)+1) // + 1 because we add "\0"
}

//export SQLGetDiagRec
func SQLGetDiagRec(
	HandleType C.SQLSMALLINT,
	Handle C.SQLHANDLE,
	RecNumber C.SQLSMALLINT,
	SQLState *C.SQLCHAR,
	NativeErrorPtr *C.SQLINTEGER,
	MessageText *C.SQLCHAR,
	BufferLength C.SQLSMALLINT,
	TextLengthPtr *C.SQLSMALLINT,
) C.SQLRETURN {
	if RecNumber != 1 {
		return C.SQL_NO_DATA
	}
	genericHandle := cgo.Handle(Handle).Value()
	var errorInfo *DriverError

	switch handle := genericHandle.(type) {
	case *environmentHandle:
		errorInfo = handle.errorInfo.errorInfo
	case *connectionHandle:
		errorInfo = handle.errorInfo.errorInfo
	case *statementHandle:
		errorInfo = handle.errorInfo.errorInfo
	default:
		panic("unknown handle type")
	}

	if errorInfo == nil {
		return C.SQL_NO_DATA
	}

	var message string
	if errorInfo.Err != nil {
		message = fmt.Sprintf("%s: %s", errorInfo.Message, errorInfo.Err)
	} else {
		message = errorInfo.Message
	}

	copyGoStringToCString(SQLState, errorInfo.SqlState, 5)
	copyGoStringToCString(MessageText, message, int(BufferLength))
	*TextLengthPtr = C.short(len(message))

	return C.SQL_SUCCESS
}

//export SQLAllocHandle
func SQLAllocHandle(HandleType C.SQLSMALLINT, InputHandle C.SQLHANDLE, OutputHandlePtr *C.SQLHANDLE) C.SQLRETURN {
	switch HandleType {
	case C.SQL_HANDLE_ENV:
		envHandle := environmentHandle{}
		envHandle.init()
		handle := cgo.NewHandle(&envHandle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
	case C.SQL_HANDLE_DBC:
		envHandle := cgo.Handle(InputHandle).Value().(*environmentHandle)
		connHandle := connectionHandle{}
		connHandle.init(envHandle)
		handle := cgo.NewHandle(&connHandle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
	case C.SQL_HANDLE_STMT:
		connHandle := cgo.Handle(InputHandle).Value().(*connectionHandle)
		stmtHandle := statementHandle{}
		stmtHandle.init(connHandle)
		handle := cgo.NewHandle(&stmtHandle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
	default:
		return C.SQL_ERROR
	}
	return C.SQL_SUCCESS
}

func maybeUnquote(literal string) string {
	literal = strings.TrimSpace(literal)

	if len(literal) < 2 {
		return literal
	}

	if literal[0] == '\'' && literal[len(literal)-1] == '\'' {
		return strings.ReplaceAll(literal[1:len(literal)-1], "''", "'")
	}

	if literal[0] == '"' && literal[len(literal)-1] == '"' {
		return strings.ReplaceAll(literal[1:len(literal)-1], "\"\"", "\"\"")
	}

	return literal
}

func splitSQL(sql string) []string {
	// XXX Should support some form of escaping
	r := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)

	return r.FindAllString(sql, -1)
}

//export SQLPrepare
func SQLPrepare(StatementHandle C.SQLHSTMT, StatementText *C.SQLCHAR, TextLength C.SQLINTEGER) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	statementText := toGoString(StatementText, TextLength)

	statementText = strings.TrimSpace(statementText)
	statementText = strings.TrimRight(statementText, ";")
	statementText = strings.TrimSpace(statementText)

	statementParts := splitSQL(statementText)
	if strings.ToUpper(statementParts[0]) != "SELECT" {
		return SetAndReturnError(s, &DriverError{SqlState: "42000", Message: fmt.Sprintf("SELECT expected, got: %s", statementParts[0])})
	}
	if strings.ToUpper(statementParts[1]) != "*" {
		return SetAndReturnError(s, &DriverError{SqlState: "42000", Message: fmt.Sprintf("* expected, got: %s", statementParts[1])})
	}
	if strings.ToUpper(statementParts[2]) != "FROM" {
		return SetAndReturnError(s, &DriverError{SqlState: "42000", Message: fmt.Sprintf("FROM expected, got: %s", statementParts[2])})
	}
	s.statement = &stmt{
		table: maybeUnquote(statementParts[3]),
	}
	if len(statementParts) > 4 {
		if strings.ToUpper(statementParts[4]) != "WHERE" {
			return SetAndReturnError(s, &DriverError{SqlState: "42000", Message: fmt.Sprintf("WHERE expected, got: %s", statementParts[4])})
		}
		if strings.ToUpper(statementParts[6]) != "=" {
			return SetAndReturnError(s, &DriverError{SqlState: "42000", Message: fmt.Sprintf("= expected, got: %s", statementParts[6])})
		}
		s.statement.condition = &cond{
			column: maybeUnquote(statementParts[5]),
			value:  maybeUnquote(statementParts[7]),
		}
	}

	return C.SQL_SUCCESS
}

//export SQLExecute
func SQLExecute(StatementHandle C.SQLHSTMT) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	s.index = -1

	if s.statement.condition == nil {
		var parts []map[string]any
		s.fetchAllParts(s.statement.table, &parts)
		s.populateColDesc(&parts)
		s.data = make([][]any, 0, len(parts))
		s.populateData(&parts)
		s.conn.updateIpnToPkMap(&parts)
	} else if s.statement.condition != nil {
		var parts []map[string]any
		var value any
		if s.params == nil {
			value = s.statement.condition.value
		} else {
			value = C.GoString((*C.char)(s.params[0].ParameterValuePtr))
		}
		if err := s.fetchPart(s.statement.table, s.statement.condition.column, value, &parts); err != nil {
			panic(err)
		}
		s.populateColDesc(&parts)
		s.data = make([][]any, 0, len(parts))
		s.populateData(&parts)
	} else {
		panic("uh?")
	}

	return C.SQL_SUCCESS
}

//export SQLNumResultCols
func SQLNumResultCols(StatementHandle C.SQLHSTMT, ColumnCountPtr *C.SQLSMALLINT) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	*ColumnCountPtr = C.short(len(s.def))

	return C.SQL_SUCCESS
}

//export SQLFreeHandle
func SQLFreeHandle(HandleType C.SQLSMALLINT, Handle C.SQLHANDLE) C.SQLRETURN {
	switch HandleType {
	case C.SQL_HANDLE_DBC:
		cgo.Handle(Handle).Delete()
	case C.SQL_HANDLE_ENV:
		cgo.Handle(Handle).Delete()
	case C.SQL_HANDLE_STMT:
		cgo.Handle(Handle).Delete()
	default:
		panic(fmt.Sprintf("Unknown handle type in call to SQLFreeHandle: %d", HandleType))
	}

	return C.SQL_SUCCESS
}

//export SQLTables
func SQLTables(StatementHandle C.SQLHSTMT, CatalogName *C.SQLCHAR, NameLength1 C.SQLSMALLINT, SchemaName *C.SQLCHAR, NameLength2 C.SQLSMALLINT, TableName *C.SQLCHAR, NameLength3 C.SQLSMALLINT, TableType *C.SQLCHAR, NameLength4 C.SQLSMALLINT) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	s.def = []*desc{
		{name: "TABLE_CAT", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_SCHEM", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_NAME", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_TYPE", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "REMARKS", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
	}
	categories := s.conn.categoryMapping
	s.index = -1
	s.data = make([][]any, 0, len(categories))
	for name := range categories {
		s.data = append(s.data, []any{nil, nil, name, "TABLE", nil})
	}

	return C.SQL_SUCCESS
}

func rowFromMap(def []*desc, data map[string]any) []any {
	row := make([]any, len(def))
	for idx, col := range def {
		if value, ok := data[col.name]; ok {
			row[idx] = value
		}
	}

	return row
}

//export SQLColumns
func SQLColumns(StatementHandle C.SQLHSTMT, CatalogName *C.SQLCHAR, NameLength1 C.SQLSMALLINT, SchemaName *C.SQLCHAR, NameLength2 C.SQLSMALLINT, TableName *C.SQLCHAR, NameLength3 C.SQLSMALLINT, ColumnName *C.SQLCHAR, NameLength4 C.SQLSMALLINT) C.SQLRETURN {
	tableName := toGoString(TableName, NameLength3)

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	s.def = []*desc{
		{name: "TABLE_CAT", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_SCHEM", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_NAME", dataType: C.SQL_VARCHAR, nullable: C.SQL_NO_NULLS},
		{name: "COLUMN_NAME", dataType: C.SQL_VARCHAR, nullable: C.SQL_NO_NULLS},
		{name: "DATA_TYPE", dataType: C.SQL_SMALLINT, nullable: C.SQL_NO_NULLS},
		{name: "TYPE_NAME", dataType: C.SQL_VARCHAR, nullable: C.SQL_NO_NULLS},
		{name: "COLUMN_SIZE", dataType: C.SQL_INTEGER, nullable: C.SQL_NULLABLE},
		{name: "BUFFER_LENGTH", dataType: C.SQL_INTEGER, nullable: C.SQL_NULLABLE},
		{name: "DECIMAL_DIGITS", dataType: C.SQL_SMALLINT, nullable: C.SQL_NULLABLE},
		{name: "NUM_PREC_RADIX ", dataType: C.SQL_SMALLINT, nullable: C.SQL_NULLABLE},
		{name: "NULLABLE", dataType: C.SQL_SMALLINT, nullable: C.SQL_NO_NULLS},
		{name: "REMARKS", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "COLUMN_DEF", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "SQL_DATA_TYPE", dataType: C.SQL_SMALLINT, nullable: C.SQL_NO_NULLS},
		{name: "SQL_DATETIME_SUB", dataType: C.SQL_SMALLINT, nullable: C.SQL_NULLABLE},
		{name: "CHAR_OCTET_LENGTH", dataType: C.SQL_INTEGER, nullable: C.SQL_NULLABLE},
		{name: "ORDINAL_POSITION", dataType: C.SQL_INTEGER, nullable: C.SQL_NO_NULLS},
		{name: "IS_NULLABLE", dataType: C.SQL_VARCHAR, nullable: C.SQL_NO_NULLS},
	}
	s.index = -1
	s.data = make([][]any, 0, 2)
	s.data = append(s.data, rowFromMap(s.def, map[string]any{
		"TABLE_NAME":    tableName,
		"COLUMN_NAME":   "IPN",
		"DATA_TYPE":     "SQL_VARCHAR",
		"TYPE_NAME":     "VARCHAR",
		"NULLABLE":      C.SQL_NO_NULLS,
		"SQL_DATA_TYPE": C.SQL_VARCHAR,
		"IS_NULLABLE":   "NO",
	}))
	s.data = append(s.data, rowFromMap(s.def, map[string]any{
		"TABLE_NAME":    tableName,
		"COLUMN_NAME":   "pk",
		"DATA_TYPE":     "SQL_VARCHAR",
		"TYPE_NAME":     "VARCHAR",
		"NULLABLE":      C.SQL_NO_NULLS,
		"SQL_DATA_TYPE": C.SQL_VARCHAR,
		"IS_NULLABLE":   "NO",
	}))

	return C.SQL_SUCCESS
}

//export SQLSetStmtAttr
func SQLSetStmtAttr(StatementHandle C.SQLHSTMT, Attribute C.SQLINTEGER, ValuePtr C.SQLPOINTER, StringLength C.SQLINTEGER) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	switch Attribute {
	case C.SQL_ATTR_ROW_ARRAY_SIZE:
		if uintptr(ValuePtr) != 1 {
			// XXX set error
			return C.SQL_ERROR
		}
		return C.SQL_SUCCESS
	case C.SQL_ATTR_ROWS_FETCHED_PTR:
		s.rowsFetchedPtr = (*C.SQLULEN)(ValuePtr)
		return C.SQL_SUCCESS
	case C.SQL_ATTR_CURSOR_TYPE:
		if uintptr(ValuePtr) != 1 {
			// XXX set error
			return C.SQL_ERROR
		}
		return C.SQL_SUCCESS
	case C.SQL_ATTR_PARAMSET_SIZE:
		if uintptr(ValuePtr) != 1 {
			// XXX set error
			return C.SQL_ERROR
		}
		return C.SQL_SUCCESS
	}
	return C.SQL_ERROR
}

//export SQLDescribeCol
func SQLDescribeCol(StatementHandle C.SQLHSTMT, ColumnNumber C.SQLUSMALLINT, ColumnName *C.SQLCHAR, BufferLength C.SQLSMALLINT,
	NameLengthPtr *C.SQLSMALLINT, DataTypePtr *C.SQLSMALLINT, ColumnSizePtr *C.SQLULEN,
	DecimalDigitsPtr *C.SQLSMALLINT, NullablePtr *C.SQLSMALLINT,
) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	if s.def == nil || s.def[ColumnNumber-1] == nil {
		return C.SQL_SUCCESS
	}

	col := s.def[ColumnNumber-1]
	copyGoStringToCString(ColumnName, col.name, len(col.name))
	*NameLengthPtr = C.short(len(col.name))
	*DataTypePtr = C.short(col.dataType)
	*NullablePtr = C.short(col.nullable)
	*ColumnSizePtr = C.SQLULEN(col.colSize)

	return C.SQL_SUCCESS
}

//export SQLBindCol
func SQLBindCol(StatementHandle C.SQLHSTMT, ColumnNumber C.SQLUSMALLINT, TargetType C.SQLSMALLINT,
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN,
) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	if s.binds == nil {
		s.binds = make([]*bind, len(s.data[0]))
	}

	s.binds[ColumnNumber-1] = &bind{
		TargetType:       TargetType,
		TargetValuePtr:   TargetValuePtr,
		BufferLength:     BufferLength,
		StrLen_or_IndPtr: StrLen_or_IndPtr,
	}

	return C.SQL_SUCCESS
}

//export SQLFetchScroll
func SQLFetchScroll(StatementHandle C.SQLHSTMT, FetchOrientation C.SQLSMALLINT, FetchOffset C.SQLLEN) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	s.index = s.index + 1

	if s.index >= len(s.data) {
		return C.SQL_NO_DATA
	}

	if s.binds != nil {
		s.populateBinds()
	}

	if s.rowsFetchedPtr != nil {
		*s.rowsFetchedPtr = 1
	}
	return C.SQL_SUCCESS
}

//export SQLFetch
func SQLFetch(StatementHandle C.SQLHSTMT) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	s.index = s.index + 1

	if s.index >= len(s.data) {
		return C.SQL_NO_DATA
	}

	if s.binds != nil {
		s.populateBinds()
	}

	if s.rowsFetchedPtr != nil {
		*s.rowsFetchedPtr = 1
	}

	return C.SQL_SUCCESS
}

func utf8stringToUTF16(str string) *[]uint16 {
	var runeValue rune
	result := make([]uint16, 0, C.SQL_MAX_MESSAGE_LENGTH)
	for _, runeValue = range str {
		result = utf16.AppendRune(result, runeValue)
	}
	return &result
}

func targetTypeToString(TargetType C.SQLSMALLINT) string {
	switch TargetType {
	case C.SQL_C_CHAR:
		return "SQL_C_CHAR"
	case C.SQL_C_WCHAR:
		return "SQL_C_WCHAR"
	case C.SQL_C_SLONG:
		return "SQL_C_SLONG"
	default:
		return fmt.Sprintf("??? (%d)", TargetType)
	}
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func populateData(value any, TargetType C.SQLSMALLINT,
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN,
) {
	switch value := value.(type) {
	case json.Number:
		switch TargetType {
		case C.SQL_C_CHAR:
			if TargetValuePtr != nil {
				dst := (*C.char)(TargetValuePtr)
				src := C.CString(value.String())
				defer C.free(unsafe.Pointer(src))
				C.strncpy(dst, src, C.size_t(BufferLength))
			}
			*StrLen_or_IndPtr = C.SQLLEN(len(value))
		case C.SQL_C_WCHAR:
			src := utf8stringToUTF16(value.String())
			length := min(C.SQLLEN(len(*src)*2), BufferLength)

			if len(*src) > 0 && TargetValuePtr != nil {
				C.memcpy(unsafe.Pointer((TargetValuePtr)), unsafe.Pointer(&(*src)[0]), C.size_t(length))
			}

			*StrLen_or_IndPtr = C.SQLLEN(length)
		case C.SQL_DOUBLE:
			if value, err := value.Float64(); err == nil {
				*(*C.double)(TargetValuePtr) = C.double(value)
			} else {
				*StrLen_or_IndPtr = C.SQL_NULL_DATA
			}
		case C.SQL_C_SBIGINT:
			if value, err := value.Int64(); err == nil {
				*(*C.long)(TargetValuePtr) = C.long(value)
			} else {
				*StrLen_or_IndPtr = C.SQL_NULL_DATA
			}
		default:
			*StrLen_or_IndPtr = C.SQL_NULL_DATA
		}
	case string:
		switch TargetType {
		case C.SQL_C_CHAR:
			if TargetValuePtr != nil {
				dst := (*C.char)(TargetValuePtr)
				src := C.CString(value)
				defer C.free(unsafe.Pointer(src))
				C.strncpy(dst, src, C.size_t(BufferLength))
			}
			*StrLen_or_IndPtr = C.SQLLEN(len(value))
		case C.SQL_C_WCHAR:
			src := utf8stringToUTF16(value)
			length := min(C.SQLLEN(len(*src)*2), BufferLength)

			if len(*src) > 0 && TargetValuePtr != nil {
				C.memcpy(unsafe.Pointer((TargetValuePtr)), unsafe.Pointer(&(*src)[0]), C.size_t(length))
			}

			*StrLen_or_IndPtr = C.SQLLEN(length)
		}
	case bool:
		switch TargetType {
		case C.SQL_C_SLONG:
			if value {
				*(*C.long)(TargetValuePtr) = 1
			} else {
				*(*C.long)(TargetValuePtr) = 0
			}
			*StrLen_or_IndPtr = 4
		case C.SQL_C_CHAR:
			value := fmt.Sprintf("%t", value)
			if TargetValuePtr != nil {
				dst := (*C.char)(TargetValuePtr)
				src := C.CString(value)
				defer C.free(unsafe.Pointer(src))
				C.strncpy(dst, src, C.size_t(BufferLength))
			}
			*StrLen_or_IndPtr = C.SQLLEN(len(value))
		default:
			*StrLen_or_IndPtr = C.SQL_NULL_DATA
		}
	case float64:
		switch TargetType {
		case C.SQL_C_CHAR:
			value := fmt.Sprintf("%g", value)
			if TargetValuePtr != nil {
				dst := (*C.char)(TargetValuePtr)
				src := C.CString(value)
				defer C.free(unsafe.Pointer(src))
				C.strncpy(dst, src, C.size_t(BufferLength))
			}
			*StrLen_or_IndPtr = C.SQLLEN(len(value))
		default:
			*StrLen_or_IndPtr = C.SQL_NULL_DATA
		}
	case nil:
		*StrLen_or_IndPtr = C.SQL_NULL_DATA
	default:
		*StrLen_or_IndPtr = C.SQL_NULL_DATA
	}
}

//export SQLGetData
func SQLGetData(StatementHandle C.SQLHSTMT, Col_or_Param_Num C.SQLUSMALLINT, TargetType C.SQLSMALLINT,
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN,
) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	populateData(s.data[s.index][Col_or_Param_Num-1], TargetType, TargetValuePtr, BufferLength, StrLen_or_IndPtr)

	return C.SQL_SUCCESS
}

//export SQLColAttribute
func SQLColAttribute(
	StatementHandle C.SQLHSTMT,
	ColumnNumber C.SQLUSMALLINT,
	FieldIdentifier C.SQLUSMALLINT,
	CharacterAttributePtr C.SQLPOINTER,
	BufferLength C.SQLSMALLINT,
	StringLengthPtr *C.SQLSMALLINT,
	NumericAttributePtr *C.SQLLEN,
) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	col := s.def[ColumnNumber-1]

	switch FieldIdentifier {
	case C.SQL_DESC_LABEL:
		copyGoStringToCString((*C.uchar)(CharacterAttributePtr), col.name, len(col.name))
		//*StringLengthPtr = C.short(len(col.name))
	default:
		if StringLengthPtr != nil {
			*StringLengthPtr = 0
		}
	}
	return C.SQL_SUCCESS
}

//export SQLRowCount
func SQLRowCount(StatementHandle C.SQLHSTMT, RowCountPtr *C.SQLLEN) C.SQLRETURN {
	*RowCountPtr = 1

	return C.SQL_SUCCESS
}

//export SQLCancel
func SQLCancel(StatementHandle C.SQLHSTMT) C.SQLRETURN {
	return C.SQL_SUCCESS
}

//export SQLFreeStmt
func SQLFreeStmt(StatementHandle C.SQLHSTMT, Option C.SQLUSMALLINT) C.SQLRETURN {
	return C.SQL_SUCCESS
}

//export SQLGetInfo
func SQLGetInfo(ConnectionHandle C.SQLHDBC, InfoType C.SQLUSMALLINT, InfoValuePtr C.SQLPOINTER,
	BufferLength C.SQLSMALLINT, StringLengthPtr *C.SQLSMALLINT,
) C.SQLRETURN {
	return C.SQL_SUCCESS
}

//export SQLDisconnect
func SQLDisconnect(ConnectionHandle C.SQLHDBC) C.SQLRETURN {
	return C.SQL_SUCCESS
}

//export SQLSetConnectAttr
func SQLSetConnectAttr(
	ConnectionHandle C.SQLHDBC,
	Attribute C.SQLINTEGER,
	ValuePtr C.SQLPOINTER,
	StringLength C.SQLINTEGER,
) C.SQLRETURN {
	return C.SQL_SUCCESS
}

//export SQLDescribeParam
func SQLDescribeParam(
	StatementHandle C.SQLHSTMT,
	ParameterNumber C.SQLUSMALLINT,
	DataTypePtr *C.SQLSMALLINT,
	ParameterSizePtr *C.SQLULEN,
	DecimalDigitsPtr *C.SQLSMALLINT,
	NullablePtr *C.SQLSMALLINT,
) C.SQLRETURN {
	if ParameterNumber != 1 {
		panic("ParameterNumber != 1")
	}

	*DataTypePtr = C.SQL_VARCHAR
	*ParameterSizePtr = 0xfffffffc // C.SQL_NO_TOTAL = -4
	*NullablePtr = C.SQL_NO_NULLS

	return C.SQL_SUCCESS
}

//export SQLBindParameter
func SQLBindParameter(
	StatementHandle C.SQLHSTMT,
	ParameterNumber C.SQLUSMALLINT,
	InputOutputType C.SQLSMALLINT,
	ValueType C.SQLSMALLINT,
	ParameterType C.SQLSMALLINT,
	ColumnSize C.SQLULEN,
	DecimalDigits C.SQLSMALLINT,
	ParameterValuePtr C.SQLPOINTER,
	BufferLength C.SQLLEN,
	StrLen_or_IndPtr *C.SQLLEN,
) C.SQLRETURN {
	if InputOutputType != C.SQL_PARAM_INPUT {
		panic("InputOutputType != C.SQL_PARAM_INPUT")
	}
	if ValueType != C.SQL_C_CHAR {
		panic("ValueType != C.SQL_C_CHAR")
	}

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	param := param{
		ValueType:         ValueType,
		ParameterValuePtr: ParameterValuePtr,
		BufferLength:      BufferLength,
	}
	s.params = append(s.params, &param)

	return C.SQL_SUCCESS
}

// SQLSetEnvAttr
