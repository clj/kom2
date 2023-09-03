package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime/cgo"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/rs/zerolog"
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

var (
	Version   string = "dev"
	Commit    string = "?"
	BuildDate string = "?"
	LogFile   string = ""
	LogFormat string = ""
	LogLevel  string = ""
)

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

func setupLogging(log zerolog.Logger, logFile, logFormat, logLevel string) zerolog.Logger {
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Printf("Error setting up logging: %s", err)
		}
		if file != nil {
			if logFormat == "pretty" {
				output := zerolog.ConsoleWriter{Out: file, TimeFormat: time.RFC3339}

				log = log.Output(output)
			} else {
				log = log.Output(file)
			}
		}
	}

	switch strings.ToLower(logLevel) {
	case "debug":
		log = log.Level(zerolog.DebugLevel)
	case "info":
		log = log.Level(zerolog.InfoLevel)
	case "warn":
		log = log.Level(zerolog.WarnLevel)
	case "error":
		log = log.Level(zerolog.ErrorLevel)
	case "fatal":
		log = log.Level(zerolog.FatalLevel)
	case "panic":
		log = log.Level(zerolog.PanicLevel)
	case "none":
		log = log.Level(zerolog.NoLevel)
	case "disabled":
		log = log.Level(zerolog.Disabled)
	case "trace":
		log = log.Level(zerolog.TraceLevel)
	default:
		log = log.Level(zerolog.InfoLevel)
	}

	return log
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

	var fetchParametersStr, fetchMetadataStr, logFile, logFormat, logLevel, httpTimeout string

	// Config file is lowest priority
	if dsn != "" {
		connHandle.inventreeConfig.server = SQLGetPrivateProfileString(dsn, "server", "", ".odbc.ini")
		connHandle.inventreeConfig.userName = SQLGetPrivateProfileString(dsn, "username", "", ".odbc.ini")
		connHandle.inventreeConfig.password = SQLGetPrivateProfileString(dsn, "password", "", ".odbc.ini")
		connHandle.inventreeConfig.apiToken = SQLGetPrivateProfileString(dsn, "apitoken", "", ".odbc.ini")
		fetchParametersStr = SQLGetPrivateProfileString(dsn, "apitoken", "", ".odbc.ini")
		fetchMetadataStr = SQLGetPrivateProfileString(dsn, "apitoken", "", ".odbc.ini")
		logFile = SQLGetPrivateProfileString(dsn, "logfile", "", ".odbc.ini")
		logFormat = SQLGetPrivateProfileString(dsn, "logformat", "", ".odbc.ini")
		logLevel = SQLGetPrivateProfileString(dsn, "loglevel", "", ".odbc.ini")

		httpTimeout = SQLGetPrivateProfileString(dsn, "httptimeout", "", ".odbc.ini")

	}

	// Then connection string is higher
	connHandle.inventreeConfig.server = conStrArg("server", connHandle.inventreeConfig.server)
	connHandle.inventreeConfig.userName = conStrArg("username", connHandle.inventreeConfig.userName)
	connHandle.inventreeConfig.password = conStrArg("password", connHandle.inventreeConfig.password)
	connHandle.inventreeConfig.apiToken = conStrArg("apitoken", connHandle.inventreeConfig.apiToken)
	fetchParametersStr = conStrArg("fetchparameters", fetchParametersStr)
	fetchMetadataStr = conStrArg("fetchparameters", fetchMetadataStr)
	logFile = conStrArg("logfile", logFile)
	logFormat = strings.ToLower(conStrArg("logformat", logFormat))
	logLevel = strings.ToLower(conStrArg("loglevel", logLevel))

	httpTimeout = strings.ToLower(conStrArg("httptimeout", httpTimeout))

	if LogFile == "" {
		connHandle.log = setupLogging(connHandle.log, logFile, logFormat, logLevel)
	}

	log := connHandle.log.With().Str("fn", "initConnection").Dict("args", zerolog.Dict().Str("dsn", dsn).Str("connectionString", connectionString).Str("userName", userName).Str("password", password)).Logger()
	log.Debug().Send()

	httpTimeoutDuration := 30 * time.Second
	if httpTimeout != "" {
		var err error
		httpTimeoutDuration, err = time.ParseDuration(httpTimeout)
		if err != nil {
			log.Error().Err(err).Msgf("Error parsing httptimeout, default timeout used: %s", httpTimeoutDuration)
		}
	}
	connHandle.httpClient = &http.Client{Timeout: httpTimeoutDuration}

	// Explicit username and password is highest
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

func resolveHandle(handle C.SQLHANDLE) any {
	defer func() { recover() }()
	return cgo.Handle(handle).Value()
}

func resolveConnectionHandle(handle C.SQLHDBC) *connectionHandle {
	defer func() { recover() }()
	return cgo.Handle(handle).Value().(*connectionHandle)
}

func resolveStatementHandle(handle C.SQLHSTMT) *statementHandle {
	defer func() { recover() }()
	return cgo.Handle(handle).Value().(*statementHandle)
}

//export SQLConnect
func SQLConnect(ConnectionHandle C.SQLHDBC,
	ServerName *C.SQLCHAR, NameLength1 C.SQLSMALLINT,
	UserName *C.SQLCHAR, NameLength2 C.SQLSMALLINT,
	Authentication *C.SQLCHAR, NameLength3 C.SQLSMALLINT,
) C.SQLRETURN {
	connHandle := resolveConnectionHandle(ConnectionHandle)
	if connHandle == nil {
		return C.SQL_INVALID_HANDLE
	}

	serverName := toGoString(ServerName, NameLength1)
	userName := toGoString(UserName, NameLength2)
	password := toGoString(Authentication, NameLength3)

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
	connHandle := resolveConnectionHandle(ConnectionHandle)
	if connHandle == nil {
		return C.SQL_INVALID_HANDLE
	}

	inConnectionString := toGoString(InConnectionString, StringLength1)

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

func getLogger(handle interface{}) zerolog.Logger {
	switch h := handle.(type) {
	case *environmentHandle:
		return zerolog.Logger{}
	case *connectionHandle:
		return h.log
	case *statementHandle:
		return h.log
	default:
		return zerolog.Logger{}
	}
}

func SetError(handle interface{}, err *DriverError) zerolog.Logger {
	switch h := handle.(type) {
	// These are all the same really:
	// case C.SQLHENV:
	// case C.SQLHDBC:
	// case C.SQLHSTMT:
	case C.SQLHANDLE:
		handle = cgo.Handle(h).Value()
	}

	switch h := handle.(type) {
	case *environmentHandle:
		h.errorInfo.errorInfo = err
		return zerolog.Logger{}
	case *connectionHandle:
		h.errorInfo.errorInfo = err
		return h.log
	case *statementHandle:
		h.errorInfo.errorInfo = err
		return h.log
	default:
		return zerolog.Logger{}
	}
}

func SetAndReturnError(handle interface{}, err *DriverError) C.SQLRETURN {
	log := SetError(handle, err)

	log.Error().Err(err).Str("return", "SQL_ERROR").Send()
	return C.SQL_ERROR
}

type errorInfo struct {
	errorInfo *DriverError
}

type logging struct {
	log zerolog.Logger
}
type environmentHandle struct {
	errorInfo
}

func (e *environmentHandle) init() {
}

func addressBytes(addr unsafe.Pointer) []byte {
	buf := make([]byte, binary.MaxVarintLen64)

	binary.BigEndian.PutUint64(buf, uint64(uintptr(addr)))
	return buf
}

func (e *environmentHandle) MarshalZerologObject(ev *zerolog.Event) {
	ev.Hex("handle_env", addressBytes(unsafe.Pointer(e)))
}

type connectionHandle struct {
	errorInfo
	logging

	httpClient *http.Client

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
	c.log = zerolog.Nop().With().Timestamp().EmbedObject(c).Logger()
	if LogFile != "" {
		c.log = setupLogging(c.log, LogFile, LogFormat, LogLevel)
	}
}

func (c *connectionHandle) MarshalZerologObject(e *zerolog.Event) {
	e.EmbedObject(c.env).Hex("handle_conn", addressBytes(unsafe.Pointer(c)))
}

func (c *connectionHandle) updateIpnToPkMap(parts *[]map[string]any) error {
	if c.ipnToPkMap == nil {
		c.ipnToPkMap = make(map[string]any)
	}

	for _, part := range *parts {
		if part["IPN"] == nil {
			continue
		}
		number, ok := part["pk"].(json.Number)
		if !ok {
			return fmt.Errorf("'pk' is not a number: %q", part["pk"])
		}
		pk, err := number.Int64()
		if err != nil {
			return fmt.Errorf("was unable to convert 'pk' to an int64: %v", part["pk"])
		}
		ipn := part["IPN"].(string)

		if ipn == "" {
			continue
		}
		c.ipnToPkMap[ipn] = pk
	}

	return nil
}

func (c *connectionHandle) getApiToken(userName, password string) (string, error) {
	request, err := http.NewRequest("GET", c.inventreeConfig.server+"/api/user/token", nil)
	if err != nil {
		return "", err
	}
	request.SetBasicAuth(userName, password)
	response, err := c.httpClient.Do(request)
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
	response, err := c.httpClient.Do(request)
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

			if err := s.conn.updateIpnToPkMap(&tmpParts); err != nil {
				return err
			}
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
		return fmt.Errorf("invalid filter column: %s", column)
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
	logging

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
	s.log = connHandle.log.With().Hex("handle_stmt", addressBytes(unsafe.Pointer(s))).Logger()
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

func copyStringToBuffer(dst *C.uchar, src string, bufferSize int) int {
	if len(src)+1 > bufferSize {
		src = src[:bufferSize-1]
	}
	cSrc := C.CString(src + "\x00")
	defer C.free(unsafe.Pointer(cSrc))
	length := len(src) + 1
	C.strncpy((*C.char)(unsafe.Pointer(dst)), cSrc, C.size_t(length))

	return length
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
	var errorInfo *DriverError
	var log zerolog.Logger

	genericHandle := resolveHandle(Handle)

	switch handle := genericHandle.(type) {
	case *environmentHandle:
		if HandleType != C.SQL_HANDLE_ENV {
			return C.SQL_INVALID_HANDLE
		}
		errorInfo = handle.errorInfo.errorInfo
		log = zerolog.Logger{}
	case *connectionHandle:
		if HandleType != C.SQL_HANDLE_DBC {
			return C.SQL_INVALID_HANDLE
		}
		errorInfo = handle.errorInfo.errorInfo
		log = handle.log.With().Str("fn", "SQLGetDiagRec").Str("handle_type", "SQL_HANDLE_DBC").Hex("handle", addressBytes(unsafe.Pointer(handle))).Logger()
	case *statementHandle:
		if HandleType != C.SQL_HANDLE_STMT {
			return C.SQL_INVALID_HANDLE
		}
		errorInfo = handle.errorInfo.errorInfo
		log = handle.log.With().Str("fn", "SQLGetDiagRec").Str("handle_type", "SQL_HANDLE_STMT").Hex("handle", addressBytes(unsafe.Pointer(handle))).Logger()

	default:
		return C.SQL_INVALID_HANDLE
	}

	/* On Windows 10 (at least, but not 11) these have to be zeroed
	   even when returning SQL_NO_DATA
	*/
	if SQLState != nil {
		*SQLState = 0
	}
	if NativeErrorPtr != nil {
		*NativeErrorPtr = 0
	}
	if MessageText != nil && BufferLength > 0 {
		*MessageText = 0
	}
	if TextLengthPtr != nil {
		*TextLengthPtr = 0
	}

	if errorInfo == nil {
		log.Debug().Msg("errInfo was nil")
		return C.SQL_NO_DATA
	}

	if RecNumber < 1 {
		log.Debug().Msgf("RecNumber:%v < 1", RecNumber)
		return C.SQL_ERROR
	}

	if RecNumber != 1 {
		log.Debug().Msgf("RecNumber:%v != 1", RecNumber)
		return C.SQL_NO_DATA
	}

	var message string
	if errorInfo.Err != nil {
		message = fmt.Sprintf("%s: %s", errorInfo.Message, errorInfo.Err)
	} else {
		message = errorInfo.Message
	}

	copyStringToBuffer(SQLState, errorInfo.SqlState, 6) // 5 + \x00
	if MessageText != nil {
		copyStringToBuffer(MessageText, message, int(BufferLength))
	}
	*TextLengthPtr = C.short(len(message))

	return C.SQL_SUCCESS
}

//export SQLGetDiagField
func SQLGetDiagField(
	HandleType C.SQLSMALLINT,
	Handle C.SQLHANDLE,
	RecNumber C.SQLSMALLINT,
	DiagIdentifier C.SQLSMALLINT,
	DiagInfoPtr C.SQLPOINTER,
	BufferLength C.SQLSMALLINT,
	StringLengthPtr *C.SQLSMALLINT,
) C.SQLRETURN {
	var errorInfo *DriverError
	var log zerolog.Logger

	genericHandle := resolveHandle(Handle)

	switch handle := genericHandle.(type) {
	case *environmentHandle:
		if HandleType != C.SQL_HANDLE_ENV {
			return C.SQL_INVALID_HANDLE
		}
		errorInfo = handle.errorInfo.errorInfo
		log = zerolog.Logger{}
	case *connectionHandle:
		if HandleType != C.SQL_HANDLE_DBC {
			return C.SQL_INVALID_HANDLE
		}
		errorInfo = handle.errorInfo.errorInfo
		log = handle.log.With().Str("fn", "SQLGetDiagField").Str("handle_type", "SQL_HANDLE_DBC").Hex("handle", addressBytes(unsafe.Pointer(handle))).Logger()
	case *statementHandle:
		if HandleType != C.SQL_HANDLE_STMT {
			return C.SQL_INVALID_HANDLE
		}
		errorInfo = handle.errorInfo.errorInfo
		log = handle.log.With().Str("fn", "SQLGetDiagField").Str("handle_type", "SQL_HANDLE_STMT").Hex("handle", addressBytes(unsafe.Pointer(handle))).Logger()

	default:
		return C.SQL_INVALID_HANDLE
	}

	if errorInfo == nil {
		log.Debug().Msg("errInfo was nil")
		return C.SQL_NO_DATA
	}

	if RecNumber < 1 {
		log.Debug().Msgf("RecNumber:%v < 1", RecNumber)
		return C.SQL_ERROR
	}

	if RecNumber != 1 {
		log.Debug().Msgf("RecNumber:%v != 1", RecNumber)
		return C.SQL_NO_DATA
	}

	copyString := func(msg string) C.SQLRETURN {
		copied := copyStringToBuffer((*C.SQLCHAR)(DiagInfoPtr), msg, int(BufferLength))
		*StringLengthPtr = C.SQLSMALLINT(copied - 1) // - \x00
		if copied-1 != len(msg) {                    // - \x00
			return C.SQL_SUCCESS_WITH_INFO
		}
		return C.SQL_SUCCESS
	}

	switch DiagIdentifier {
	case C.SQL_DIAG_NUMBER:
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_NUMBER").Int("DiagInfoPtr", 1).Str("return", "SQL_SUCCESS").Send()
		*((*C.SQLINTEGER)(DiagInfoPtr)) = 1
		return C.SQL_SUCCESS
	case C.SQL_DIAG_NATIVE:
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_NATIVE").Int("DiagInfoPtr", 0).Str("return", "SQL_SUCCESS").Send()
		*((*C.SQLINTEGER)(DiagInfoPtr)) = 0
		return C.SQL_SUCCESS
	case C.SQL_DIAG_SQLSTATE:
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_SQLSTATE").Str("DiagInfoPtr", errorInfo.SqlState).Str("return", "SQL_SUCCESS").Send()
		return copyString(errorInfo.SqlState)
	case C.SQL_DIAG_MESSAGE_TEXT:
		var message string
		if errorInfo.Err != nil {
			message = fmt.Sprintf("%s: %s", errorInfo.Message, errorInfo.Err)
		} else {
			message = errorInfo.Message
		}
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_MESSAGE_TEXT").Str("DiagInfoPtr", message).Str("return", "SQL_SUCCESS").Send()
		return copyString(message)
	case C.SQL_DIAG_CLASS_ORIGIN:
		str := "ISO 9075"
		if strings.HasPrefix(errorInfo.SqlState, "IM") {
			str = "ODBC 3.0"
		}
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_CLASS_ORIGIN").Str("DiagInfoPtr", str).Str("return", "SQL_SUCCESS").Send()
		return copyString(str)
	case C.SQL_DIAG_SUBCLASS_ORIGIN:
		// Logic taken from the SQLite ODBC driver, this looks different from the docs:
		//  https://learn.microsoft.com/en-us/sql/odbc/reference/syntax/sqlgetdiagfield-function?view=sql-server-ver16
		str := "ISO 9075"
		if strings.HasPrefix(errorInfo.SqlState, "IM") || strings.HasPrefix(errorInfo.SqlState, "HY") || errorInfo.SqlState == "2" || errorInfo.SqlState == "0" || errorInfo.SqlState == "4" {
			str = "ODBC 3.0"
		}
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_SUBCLASS_ORIGIN").Str("DiagInfoPtr", str).Str("return", "SQL_SUCCESS").Send()
		return copyString(str)
	case C.SQL_DIAG_CONNECTION_NAME:
		str := "kom2" // FIXME: use dsn?
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_CONNECTION_NAME").Str("DiagInfoPtr", str).Str("return", "SQL_SUCCESS").Send()
		return copyString(str)
	case C.SQL_DIAG_SERVER_NAME:
		str := "inventree" // FIXME: actual configured server
		log.Debug().Str("DiagIdentifier", "SQL_DIAG_SERVER_NAME").Str("DiagInfoPtr", str).Str("return", "SQL_SUCCESS").Send()
		return copyString(str)
	}

	log.Error().Int("DiagIdentifier", int(DiagIdentifier)).Str("return", "SQL_ERROR").Send()

	return C.SQL_ERROR
}

//export SQLAllocHandle
func SQLAllocHandle(HandleType C.SQLSMALLINT, InputHandle C.SQLHANDLE, OutputHandlePtr *C.SQLHANDLE) (ret C.SQLRETURN) {
	var log zerolog.Logger

	setOutputHandleOnError := func(nullHandle C.SQLHANDLE) {
		if err := recover(); err != nil {
			if OutputHandlePtr != nil {
				*OutputHandlePtr = nullHandle
			}
			log.Info().Str("return", "SQL_ERROR").Send()
			ret = C.SQL_ERROR
		}
	}
	setErrorInfoOnError := func(handle interface{}, msg string) {
		if err := recover(); err != nil {
			SetError(handle, &DriverError{SqlState: "HY000", Message: msg, Err: err.(error)})
		}
	}

	switch HandleType {
	case C.SQL_HANDLE_ENV:
		envHandle := environmentHandle{}
		envHandle.init()
		handle := cgo.NewHandle(&envHandle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
	case C.SQL_HANDLE_DBC:
		defer setOutputHandleOnError(C.SQLHANDLE(uintptr(C.SQL_NULL_HDBC)))
		envHandle := cgo.Handle(InputHandle).Value().(*environmentHandle)
		defer setErrorInfoOnError(envHandle, "Error initialising connection handle")
		connHandle := connectionHandle{}
		connHandle.init(envHandle)
		handle := cgo.NewHandle(&connHandle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
		log = connHandle.log.With().Str("fn", "SQLAllocHandle").Str("handle_type", "SQL_HANDLE_DBC").Hex("handle", addressBytes(unsafe.Pointer(handle))).Logger()
	case C.SQL_HANDLE_STMT:
		defer setOutputHandleOnError(C.SQLHANDLE(uintptr(C.SQL_NULL_HSTMT)))
		connHandle := cgo.Handle(InputHandle).Value().(*connectionHandle)
		defer setErrorInfoOnError(connHandle, "Error initialising statement handle")
		stmtHandle := statementHandle{}
		stmtHandle.init(connHandle)
		handle := cgo.NewHandle(&stmtHandle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
		log = stmtHandle.log.With().Str("fn", "SQLAllocHandle").Str("handle_type", "SQL_HANDLE_STMT").Hex("handle", addressBytes(unsafe.Pointer(handle))).Logger()
	default:
		log.Info().Str("return", "SQL_ERROR").Send()
		return C.SQL_ERROR
	}

	log.Info().Str("return", "SQL_SUCCESS").Send()
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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	s.index = -1

	if s.statement.condition == nil {
		var parts []map[string]any
		if err := s.fetchAllParts(s.statement.table, &parts); err != nil {
			return SetAndReturnError(s, &DriverError{SqlState: "HY000", Message: "Unable to fetch parts", Err: err})
		}
		s.populateColDesc(&parts)
		s.data = make([][]any, 0, len(parts))
		s.populateData(&parts)
		if err := s.conn.updateIpnToPkMap(&parts); err != nil {
			return SetAndReturnError(s, &DriverError{SqlState: "HY000", Message: "Unable to fetch parts", Err: err})
		}
	} else {
		var parts []map[string]any
		var value any
		if s.params == nil {
			value = s.statement.condition.value
		} else {
			value = C.GoString((*C.char)(s.params[0].ParameterValuePtr))
		}
		if err := s.fetchPart(s.statement.table, s.statement.condition.column, value, &parts); err != nil {
			return SetAndReturnError(s, &DriverError{SqlState: "HY000", Message: "Unable to fetch parts", Err: err})
		}
		s.populateColDesc(&parts)
		s.data = make([][]any, 0, len(parts))
		s.populateData(&parts)
	}

	return C.SQL_SUCCESS
}

//export SQLNumResultCols
func SQLNumResultCols(StatementHandle C.SQLHSTMT, ColumnCountPtr *C.SQLSMALLINT) C.SQLRETURN {
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	*ColumnCountPtr = C.short(len(s.def))

	return C.SQL_SUCCESS
}

func recoverFromInvalidHandle(ret *C.SQLRETURN) {
	if err := recover(); err != nil {
		*ret = C.SQL_INVALID_HANDLE
	}
}

//export SQLFreeHandle
func SQLFreeHandle(HandleType C.SQLSMALLINT, Handle C.SQLHANDLE) (ret C.SQLRETURN) {
	defer recoverFromInvalidHandle(&ret)

	switch HandleType {
	case C.SQL_HANDLE_DBC:
		cgo.Handle(Handle).Delete()
	case C.SQL_HANDLE_ENV:
		cgo.Handle(Handle).Delete()
	case C.SQL_HANDLE_STMT:
		cgo.Handle(Handle).Delete()
	default:
		return C.SQL_INVALID_HANDLE
	}

	return C.SQL_SUCCESS
}

//export SQLTables
func SQLTables(StatementHandle C.SQLHSTMT, CatalogName *C.SQLCHAR, NameLength1 C.SQLSMALLINT, SchemaName *C.SQLCHAR, NameLength2 C.SQLSMALLINT, TableName *C.SQLCHAR, NameLength3 C.SQLSMALLINT, TableType *C.SQLCHAR, NameLength4 C.SQLSMALLINT) C.SQLRETURN {
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	log := s.log.With().Str("fn", "SQLTables").Logger()

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
		log.Debug().Str("category", name).Msg("adding category")
	}
	log.Debug().Msgf("added %d categories", len(s.data))

	log.Info().Str("return", "SQL_SUCCESS").Send()
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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	tableName := toGoString(TableName, NameLength3)

	log := s.log.With().Str("fn", "SQLColumns").Dict("args", zerolog.Dict().Str("TableName", tableName)).Logger()

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

	log.Info().Str("return", "SQL_SUCCESS").Send()
	return C.SQL_SUCCESS
}

//export SQLSetStmtAttr
func SQLSetStmtAttr(StatementHandle C.SQLHSTMT, Attribute C.SQLINTEGER, ValuePtr C.SQLPOINTER, StringLength C.SQLINTEGER) C.SQLRETURN {
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	log := s.log.With().Str("fn", "SQLSetStmtAttr").Dict("args", zerolog.Dict().Int("Attribute", int(Attribute)).Hex("ValuePtr", addressBytes(unsafe.Pointer(ValuePtr))).Int("StringLength", int(StringLength))).Logger()

	switch Attribute {
	case C.SQL_ATTR_ROW_ARRAY_SIZE:
		if uintptr(ValuePtr) != 1 {
			// XXX set error
			log.Info().Str("return", "SQL_ERROR").Send()
			return C.SQL_ERROR
		}
		log.Info().Str("return", "SQL_SUCCESS").Send()
		return C.SQL_SUCCESS
	case C.SQL_ATTR_ROWS_FETCHED_PTR:
		s.rowsFetchedPtr = (*C.SQLULEN)(ValuePtr)
		log.Debug().Msg("set rowsFetchedPtr")
		log.Info().Str("return", "SQL_ERROR").Send()
		return C.SQL_SUCCESS
	case C.SQL_ATTR_CURSOR_TYPE:
		if uintptr(ValuePtr) != 1 {
			// XXX set error
			log.Info().Str("return", "SQL_ERROR").Send()
			return C.SQL_ERROR
		}
		log.Info().Str("return", "SQL_ERROR").Send()
		return C.SQL_SUCCESS
	case C.SQL_ATTR_PARAMSET_SIZE:
		if uintptr(ValuePtr) != 1 {
			// XXX set error
			log.Info().Str("return", "SQL_ERROR").Send()
			return C.SQL_ERROR
		}
		log.Info().Str("return", "SQL_ERROR").Send()
		return C.SQL_SUCCESS
	}
	log.Info().Str("return", "SQL_ERROR").Send()
	return C.SQL_ERROR
}

//export SQLDescribeCol
func SQLDescribeCol(StatementHandle C.SQLHSTMT, ColumnNumber C.SQLUSMALLINT, ColumnName *C.SQLCHAR, BufferLength C.SQLSMALLINT,
	NameLengthPtr *C.SQLSMALLINT, DataTypePtr *C.SQLSMALLINT, ColumnSizePtr *C.SQLULEN,
	DecimalDigitsPtr *C.SQLSMALLINT, NullablePtr *C.SQLSMALLINT,
) C.SQLRETURN {
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	log := s.log.With().Str("fn", "SQLBindCol").Dict("args", zerolog.Dict().Uint("ColumnNumber", uint(ColumnNumber))).Logger()

	if s.binds == nil {
		length := len(s.data[0])
		s.binds = make([]*bind, length)
		log.Debug().Int("len", length).Msg("creating binds array")
	}

	s.binds[ColumnNumber-1] = &bind{
		TargetType:       TargetType,
		TargetValuePtr:   TargetValuePtr,
		BufferLength:     BufferLength,
		StrLen_or_IndPtr: StrLen_or_IndPtr,
	}

	log.Info().Str("return", "SQL_SUCCESS").Send()
	return C.SQL_SUCCESS
}

//export SQLFetchScroll
func SQLFetchScroll(StatementHandle C.SQLHSTMT, FetchOrientation C.SQLSMALLINT, FetchOffset C.SQLLEN) C.SQLRETURN {
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	s.index = s.index + 1
	log := s.log.With().Str("fn", "SQLFetchScroll").Int("index", s.index).Logger()

	if s.index >= len(s.data) {
		log.Info().Str("return", "SQL_NO_DATA").Send()
		return C.SQL_NO_DATA
	}

	if s.binds != nil {
		log.Debug().Msg("populating binds")
		s.populateBinds()
	}

	if s.rowsFetchedPtr != nil {
		log.Debug().Msg("setting rowsFetchedPtr")
		*s.rowsFetchedPtr = 1
	}

	log.Info().Str("return", "SQL_SUCCESS").Send()
	return C.SQL_SUCCESS
}

//export SQLFetch
func SQLFetch(StatementHandle C.SQLHSTMT) C.SQLRETURN {
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	s.index = s.index + 1
	log := s.log.With().Str("fn", "SQLFetch").Int("index", s.index).Logger()

	if s.index >= len(s.data) {
		log.Info().Str("return", "SQL_NO_DATA").Send()
		return C.SQL_NO_DATA
	}

	if s.binds != nil {
		log.Debug().Msg("populating binds")
		s.populateBinds()
	}

	if s.rowsFetchedPtr != nil {
		log.Debug().Msg("setting rowsFetchedPtr")
		*s.rowsFetchedPtr = 1
	}

	log.Info().Str("return", "SQL_SUCCESS").Send()
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

// See: https://learn.microsoft.com/en-us/sql/odbc/reference/appendixes/converting-data-from-sql-to-c-data-types?view=sql-server-ver16
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
			var strValue string
			if value {
				strValue = "1"
			} else {
				strValue = "0"
			}
			if TargetValuePtr != nil {
				dst := (*C.char)(TargetValuePtr)
				src := C.CString(strValue)
				defer C.free(unsafe.Pointer(src))
				C.strncpy(dst, src, C.size_t(BufferLength))
			}
			*StrLen_or_IndPtr = C.SQLLEN(len(strValue))
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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}
	log := s.log.With().Str("fn", "SQLGetData").Dict("args", zerolog.Dict().Uint("Col_or_Param_Num", uint(Col_or_Param_Num))).Int("index", s.index).Logger()

	populateData(s.data[s.index][Col_or_Param_Num-1], TargetType, TargetValuePtr, BufferLength, StrLen_or_IndPtr)

	log.Info().Str("return", "SQL_SUCCESS").Send()
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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

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
	c := resolveConnectionHandle(ConnectionHandle)
	if c == nil {
		return C.SQL_INVALID_HANDLE
	}

	log := c.log.With().Str("fn", "SQLGetInfo").Dict("args", zerolog.Dict().Uint("InfoType", uint(InfoType)).Hex("InfoValuePtr", addressBytes(unsafe.Pointer(InfoValuePtr)))).Logger()

	returnString := func(str string) {
		if len(str) >= int(BufferLength) {
			str = str[:BufferLength-1]
		}
		if InfoValuePtr != nil {
			dst := (*C.char)(InfoValuePtr)
			src := C.CString(str + "\x00")
			defer C.free(unsafe.Pointer(src))
			C.strncpy(dst, src, C.size_t(len(str)+1))
		}
		if StringLengthPtr != nil {
			*StringLengthPtr = C.SQLSMALLINT(len(str))
		}
	}

	switch InfoType {
	case C.SQL_DRIVER_ODBC_VER:
		returnString("03.00")
	case C.SQL_IDENTIFIER_QUOTE_CHAR:
		returnString("\"")
	case C.SQL_GETDATA_EXTENSIONS:
		*((*C.SQLUINTEGER)(InfoValuePtr)) = C.SQL_GD_ANY_COLUMN | C.SQL_GD_ANY_ORDER | C.SQL_GD_BOUND
	case C.SQL_DRIVER_NAME:
		returnString("kom2")
	case C.SQL_DRIVER_VER:
		returnString(fmt.Sprintf("%s %s %s", Version, Commit, BuildDate))
	case C.SQL_TXN_CAPABLE:
		*((*C.SQLUINTEGER)(InfoValuePtr)) = C.SQL_TC_NONE
	default:
		log.Info().Str("return", "SQL_ERROR").Send()
		return C.SQL_ERROR
	}

	log.Info().Str("return", "SQL_SUCCESS").Send()
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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	if ParameterNumber != 1 {
		return SetAndReturnError(s, &DriverError{SqlState: "07009", Message: "ParameterNumber != 1"})
	}

	*DataTypePtr = C.SQL_VARCHAR
	*ParameterSizePtr = 0xfffffffc // C.SQL_NO_TOTAL = -4  FIXME: Is this correct?
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
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	if InputOutputType != C.SQL_PARAM_INPUT {
		return SetAndReturnError(s, &DriverError{SqlState: "HYC00", Message: "InputOutputType != C.SQL_PARAM_INPUT"})
	}
	if ValueType != C.SQL_C_CHAR {
		return SetAndReturnError(s, &DriverError{SqlState: "HYC00", Message: "ValueType != C.SQL_C_CHAR"})
	}

	param := param{
		ValueType:         ValueType,
		ParameterValuePtr: ParameterValuePtr,
		BufferLength:      BufferLength,
	}
	s.params = append(s.params, &param)

	return C.SQL_SUCCESS
}

//export SQLSetEnvAttr
func SQLSetEnvAttr(EnvironmentHandle C.SQLHENV, Attribute C.SQLINTEGER, ValuePtr C.SQLPOINTER, StringLength C.SQLINTEGER) C.SQLRETURN {
	switch Attribute {
	case C.SQL_ATTR_ODBC_VERSION:
		if int(uintptr(ValuePtr)) != C.SQL_OV_ODBC3 {
			return SetAndReturnError(EnvironmentHandle, &DriverError{SqlState: "HY024", Message: "Unsupported value for ODBC version"})
		}
		return C.SQL_SUCCESS
	default:
		return SetAndReturnError(EnvironmentHandle, &DriverError{SqlState: "HYC00", Message: "Unsupported attribute"})
	}
}

//export SQLGetStmtAttr
func SQLGetStmtAttr(
	StatementHandle C.SQLHSTMT,
	Attribute C.SQLINTEGER,
	ValuePtr C.SQLPOINTER,
	BufferLength C.SQLINTEGER,
	StringLengthPtr *C.SQLINTEGER,
) C.SQLRETURN {
	s := resolveStatementHandle(StatementHandle)
	if s == nil {
		return C.SQL_INVALID_HANDLE
	}

	log := s.log.With().Str("fn", "SQLGetStmtAttr").Dict("args", zerolog.Dict().Uint("Attribute", uint(Attribute))).Logger()

	switch Attribute {
	case C.SQL_ATTR_IMP_ROW_DESC:
		fallthrough
	case C.SQL_ATTR_APP_ROW_DESC:
		fallthrough
	case C.SQL_ATTR_IMP_PARAM_DESC:
		fallthrough
	case C.SQL_ATTR_APP_PARAM_DESC:
		value := C.SQLHDESC(uintptr(0xdeadbeef))
		*((*C.SQLHDESC)(ValuePtr)) = value
		if StringLengthPtr != nil {
			*StringLengthPtr = C.SQLINTEGER(unsafe.Sizeof(value))
		}
		log.Info().Str("return", "SQL_SUCCESS").Send()
		return C.SQL_SUCCESS
	}

	log.Info().Str("return", "SQL_ERROR").Send()
	return C.SQL_ERROR
}

//export SQLEndTran
func SQLEndTran(
	HandleType C.SQLSMALLINT,
	Handle C.SQLHANDLE,
	CompletionType C.SQLSMALLINT,
) C.SQLRETURN {
	return C.SQL_SUCCESS
}

//export VersionInfo
func VersionInfo(
	ResultPtr *C.char,
	BufferLength C.int,
) int {
	ver := fmt.Sprintf("%s %s %s", Version, Commit, BuildDate)

	if ResultPtr != nil && BufferLength > 1 {
		dst := (*C.char)(ResultPtr)
		src := C.CString(ver + "\x00")
		defer C.free(unsafe.Pointer(src))
		length := min(len(ver)+1, int(BufferLength)-1)
		C.strncpy(dst, src, C.size_t(length))
	}

	return len(ver) + 1
}
