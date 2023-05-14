package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/cgo"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unsafe"
)

// #cgo LDFLAGS: -lodbcinst
// #include <stdlib.h>
// #include <string.h>
// #include <sqltypes.h>
// #include <sql.h>
// #include <sqlext.h>
// #include <odbcinst.h>
import "C"

func SQLGetPrivateProfileString(section, entry, defaultValue, filename string) string {
	cSection := C.CString(section)
	defer C.free(unsafe.Pointer(cSection))
	cEntry := C.CString(entry)
	defer C.free(unsafe.Pointer(cEntry))
	cDefaultValue := C.CString(defaultValue)
	defer C.free(unsafe.Pointer(cDefaultValue))
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	buffer := (*C.char)(C.malloc(C.SQL_MAX_MESSAGE_LENGTH))
	defer C.free(unsafe.Pointer(buffer))

	C.SQLGetPrivateProfileString(cSection, cEntry, cDefaultValue, buffer, C.SQL_MAX_MESSAGE_LENGTH, cFilename)

	return C.GoString(buffer)
}

func toGoString[T C.int | C.short](str *C.SQLCHAR, len T) string {
	if str == nil {
		return ""
	}
	if len == C.SQL_NTS {
		return C.GoString((*C.char)(unsafe.Pointer(str)))
	}
	return C.GoStringN((*C.char)(unsafe.Pointer(str)), C.int(len))
}

func (connHandle *connectionHandle) initConnection(dsn string) C.SQLRETURN {
	// Todo: also pull from the DSN/serverName
	connHandle.inventreeConfig.server = SQLGetPrivateProfileString(dsn, "server", "", ".odbc.ini")
	connHandle.inventreeConfig.apiToken = SQLGetPrivateProfileString(dsn, "password", "", ".odbc.ini")

	// if userName := toGoString(UserName, NameLength2); username != "" {
	// 	connHandle.inventreeConfig.user = userName
	// }
	// if authentication := toGoString(Authentication, NameLength3); authentication != "" {
	// 	connHandle.inventreeConfig.apiToken = authentication
	// }

	if connHandle.inventreeConfig.server == "" {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "No server specified"})
	}
	if connHandle.inventreeConfig.apiToken == "" {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "No API token (Password) specified"})
	}

	fmt.Println("WAHT???")
	if err := connHandle.updateCategoryMapping(); err != nil {
		fmt.Println("oh???")

		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "Error updating category list", Err: err})
	}
	fmt.Printf("%+q\n", connHandle.categoryMapping)

	return C.SQL_SUCCESS
}

//export SQLConnect
func SQLConnect(ConnectionHandle C.SQLHDBC,
	ServerName *C.SQLCHAR, NameLength1 C.SQLSMALLINT,
	UserName *C.SQLCHAR, NameLength2 C.SQLSMALLINT,
	Authentication *C.SQLCHAR, NameLength3 C.SQLSMALLINT) C.SQLRETURN {

	fmt.Printf("%v %d %v %d %v %d\n", ServerName, NameLength1, UserName, NameLength2, Authentication, NameLength3)

	serverName := toGoString(ServerName, NameLength1)
	connHandle := cgo.Handle(ConnectionHandle).Value().(*connectionHandle)
	connHandle.initConnection(serverName)

	return C.SQL_SUCCESS
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
	DriverCompletion C.SQLUSMALLINT) C.SQLRETURN {

	inConnectionString := toGoString(InConnectionString, StringLength1)
	fmt.Printf("SQLGSQLDriverConnectetInfo %+q\n", inConnectionString)

	args := make(map[string]string)
	for _, arg := range strings.Split(inConnectionString, ";") {
		sarg := strings.SplitN(arg, "=", 2)
		if len(sarg) == 1 {
			args[strings.ToLower(sarg[0])] = ""
		} else {
			args[strings.ToLower(sarg[0])] = sarg[1]
		}
	}
	connHandle := cgo.Handle(ConnectionHandle).Value().(*connectionHandle)
	connHandle.initConnection(args["dsn"]) // FIXME

	return C.SQL_SUCCESS
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
		server string
		//userName string
		apiToken string
	}
	categoryMapping map[string]int
}

func (c *connectionHandle) init(envHandle *environmentHandle) {
	c.env = envHandle
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
	var categories = []category{}
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
	args["category"] = strconv.Itoa(s.conn.categoryMapping[category])

	return s.conn.apiGet("/api/part/", args, parts)
}

func (s *statementHandle) populateColDesc(data *[]map[string]any) {
	columns := make(map[string]*desc)

	for _, row := range *data {
		for key, _ := range row {
			if _, ok := columns[key]; !ok {
				var dataType C.short
				switch row[key].(type) {
				case string:
					dataType = C.SQL_VARCHAR
				case float64:
					dataType = C.SQL_DOUBLE
				case bool:
					dataType = C.SQL_INTEGER
				}
				columns[key] = &desc{name: key, dataType: dataType, nullable: C.SQL_NULLABLE}
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

type desc struct {
	name          string
	dataType      C.short
	colSize       int
	decimalDigits int
	nullable      int
}

type statementHandle struct {
	errorInfo

	conn        *connectionHandle
	columnNames []string
	def         []*desc
	binds       []*bind
	data        [][]any
	index       int
}

func (s *statementHandle) init(connHandle *connectionHandle) {
	s.conn = connHandle
	s.index = -1
}

func (s *statementHandle) populateBinds() {
	for idx, bind := range s.binds {
		if bind == nil {
			//fmt.Println("phhhhh\n")
			continue
		}
		//fmt.Println("%d\n", idx)

		value := s.data[s.index][idx]
		populateData(value, bind.TargetType, bind.TargetValuePtr, bind.BufferLength, bind.StrLen_or_IndPtr)
	}
}

func copyGoStringToCString(dst *C.uchar, src string, length int) {
	cSrc := C.CString(src + "\x00")
	defer C.free(unsafe.Pointer(cSrc))
	C.strncpy((*C.char)(unsafe.Pointer(dst)), cSrc, C.ulong(length)+1) // + 1 because we add "\0"
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
	TextLengthPtr *C.SQLSMALLINT) C.SQLRETURN {

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

	copyGoStringToCString(SQLState, errorInfo.SqlState, 5)
	copyGoStringToCString(MessageText, errorInfo.Message, int(BufferLength))
	*TextLengthPtr = C.short(len(errorInfo.Message))

	return C.SQL_SUCCESS
}

//export SQLAllocHandle
func SQLAllocHandle(HandleType C.SQLSMALLINT, InputHandle C.SQLHANDLE, OutputHandlePtr *C.SQLHANDLE) C.SQLRETURN {
	fmt.Printf("SQLAllocHandle(%d)\n", HandleType)
	switch HandleType {
	case C.SQL_HANDLE_ENV:
		envHandle := environmentHandle{}
		envHandle.init()
		handle := cgo.NewHandle(&envHandle)
		fmt.Printf("SQLAllocHandle(SQL_HANDLE_ENV)\n")
		fmt.Printf("   envHandle: %+v, handle: %d\n", envHandle, handle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
	case C.SQL_HANDLE_DBC:
		envHandle := cgo.Handle(InputHandle).Value().(*environmentHandle)
		connHandle := connectionHandle{}
		connHandle.init(envHandle)
		handle := cgo.NewHandle(&connHandle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
		fmt.Printf("SQLAllocHandle(SQL_HANDLE_DBC)\n")
		fmt.Printf("   envHandle: %+v, handle: %d\n", envHandle, handle)
	case C.SQL_HANDLE_STMT:
		connHandle := cgo.Handle(InputHandle).Value().(*connectionHandle)
		stmtHandle := statementHandle{}
		stmtHandle.init(connHandle)
		handle := cgo.NewHandle(&stmtHandle)
		fmt.Printf("SQLAllocHandle(SQL_HANDLE_STMT)\n")
		fmt.Printf("   connHandle: %+v, handle: %d\n", connHandle, handle)
		*OutputHandlePtr = C.SQLHANDLE(unsafe.Pointer(handle))
	default:
		return C.SQL_ERROR
	}
	return C.SQL_SUCCESS
}

//export SQLPrepare
func SQLPrepare(StatementHandle C.SQLHSTMT, StatementText *C.SQLCHAR, TextLength C.SQLINTEGER) C.SQLRETURN {
	statementText := toGoString(StatementText, TextLength)

	fmt.Printf("StatementText: %s\n", statementText)

	return C.SQL_SUCCESS
}

//export SQLExecute
func SQLExecute(StatementHandle C.SQLHSTMT) C.SQLRETURN {
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	s.index = -1

	var parts []map[string]any
	s.fetchAllParts("Resistors", &parts)
	s.populateColDesc(&parts)
	s.populateData(&parts)

	fmt.Printf("XXX: %d %d", len(parts), len(s.data))
	//fmt.Printf("XXX: %+q", s.data)
	//fmt.Printf("XXX: %+q", s.columnNames)
	//fmt.Printf("XXX: %+q", s.def)

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
	return C.SQL_SUCCESS
}

//export SQLTables
func SQLTables(StatementHandle C.SQLHSTMT, CatalogName *C.SQLCHAR, NameLength1 C.SQLSMALLINT, SchemaName *C.SQLCHAR, NameLength2 C.SQLSMALLINT, TableName *C.SQLCHAR, NameLength3 C.SQLSMALLINT, TableType *C.SQLCHAR, NameLength4 C.SQLSMALLINT) C.SQLRETURN {
	catalogName := toGoString(CatalogName, NameLength1)
	schemaName := toGoString(SchemaName, NameLength1)
	tableName := toGoString(TableName, NameLength1)
	tableType := toGoString(TableType, NameLength1)

	fmt.Printf("SQLTables %q  %q %q  %q\n", catalogName, schemaName, tableName, tableType)
	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	s.def = []*desc{
		{name: "TABLE_CAT", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_SCHEM", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_NAME", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "TABLE_TYPE", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
		{name: "REMARKS", dataType: C.SQL_VARCHAR, nullable: C.SQL_NULLABLE},
	}
	categories := s.conn.categoryMapping
	for name, _ := range categories {
		s.data = append(s.data, []any{nil, nil, name, "TABLE", nil})
	}

	return C.SQL_SUCCESS
}

//export SQLSetStmtAttr
func SQLSetStmtAttr(StatementHandle C.SQLHSTMT, Attribute C.SQLINTEGER, ValuePtr C.SQLPOINTER, StringLength C.SQLINTEGER) C.SQLRETURN {
	fmt.Printf("SQLSetStmtAttr %q  %q %q  %q\n", StatementHandle, Attribute, ValuePtr, StringLength)

	return C.SQL_SUCCESS

}

//export SQLDescribeCol
func SQLDescribeCol(StatementHandle C.SQLHSTMT, ColumnNumber C.SQLUSMALLINT, ColumnName *C.SQLCHAR, BufferLength C.SQLSMALLINT,
	NameLengthPtr *C.SQLSMALLINT, DataTypePtr *C.SQLSMALLINT, ColumnSizePtr *C.SQLULEN,
	DecimalDigitsPtr *C.SQLSMALLINT, NullablePtr *C.SQLSMALLINT) C.SQLRETURN {
	fmt.Printf("SQLDescribeCol %q  %q %v %q %v %v %v %v %v\n", StatementHandle, ColumnNumber, ColumnName, BufferLength,
		NameLengthPtr, DataTypePtr, ColumnSizePtr, DecimalDigitsPtr, NullablePtr)

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	if s.def == nil || s.def[ColumnNumber-1] == nil {
		return C.SQL_SUCCESS
	}

	col := s.def[ColumnNumber-1]
	copyGoStringToCString(ColumnName, col.name, len(col.name))
	*NameLengthPtr = C.short(len(col.name))
	*DataTypePtr = C.short(col.dataType)
	*NullablePtr = C.short(col.nullable)

	return C.SQL_SUCCESS
}

//export SQLBindCol
func SQLBindCol(StatementHandle C.SQLHSTMT, ColumnNumber C.SQLUSMALLINT, TargetType C.SQLSMALLINT,
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN) C.SQLRETURN {

	fmt.Printf("SQLBindCol %q  %q %v %q %v %v\n", StatementHandle, ColumnNumber, TargetType, TargetValuePtr,
		BufferLength, StrLen_or_IndPtr)

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
	fmt.Printf("SQLFetchScroll %q  %q %q\n", StatementHandle, FetchOrientation, FetchOffset)

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	s.index = s.index + 1

	//fmt.Printf("  index %q\n", s.index)
	//fmt.Printf("  data len %q\n", len(s.data))

	if s.index >= len(s.data) {
		return C.SQL_NO_DATA
	}

	if s.binds != nil {
		s.populateBinds()
	}

	return C.SQL_SUCCESS
}

//export SQLFetch
func SQLFetch(StatementHandle C.SQLHSTMT) C.SQLRETURN {
	//fmt.Printf("SQLFetch %q\n", StatementHandle)

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	s.index = s.index + 1

	//fmt.Printf("  index %q\n", s.index)
	//fmt.Printf("  data len %q\n", len(s.data))

	if s.index >= len(s.data) {
		return C.SQL_NO_DATA
	}

	if s.binds != nil {
		s.populateBinds()
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

func populateData(value any, TargetType C.SQLSMALLINT,
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN) {

	//fmt.Printf("Type: %T TargetType: %d %s \n", value, TargetType, targetTypeToString(TargetType))
	switch value := value.(type) {
	case string:
		switch TargetType {
		case C.SQL_C_CHAR:
			dst := (*C.char)(TargetValuePtr)
			src := C.CString(value)
			defer C.free(unsafe.Pointer(src))
			C.strncpy(dst, src, C.ulong(BufferLength))
			*StrLen_or_IndPtr = C.long(len(value))
		case C.SQL_C_WCHAR:
			src := utf8stringToUTF16(value)
			C.memcpy(unsafe.Pointer((TargetValuePtr)), unsafe.Pointer(&(*src)[0]), C.ulong(len(*src)*2))
			*StrLen_or_IndPtr = C.long(len(*src) * 2)
		}
	case bool:
		switch TargetType {
		case C.SQL_C_SLONG:
			if value {
				*(*C.int32_t)(TargetValuePtr) = 1
			} else {
				*(*C.int32_t)(TargetValuePtr) = 0
			}
			*StrLen_or_IndPtr = 4
		case C.SQL_C_CHAR:
			value := fmt.Sprintf("%t", value)
			dst := (*C.char)(TargetValuePtr)
			src := C.CString(value)
			defer C.free(unsafe.Pointer(src))
			C.strncpy(dst, src, C.ulong(BufferLength))
			*StrLen_or_IndPtr = C.long(len(value))
		default:
			*StrLen_or_IndPtr = C.SQL_NULL_DATA
		}
	case float64:
		switch TargetType {
		case C.SQL_C_CHAR:
			value := fmt.Sprintf("%g", value)
			dst := (*C.char)(TargetValuePtr)
			src := C.CString(value)
			defer C.free(unsafe.Pointer(src))
			C.strncpy(dst, src, C.ulong(BufferLength))
			*StrLen_or_IndPtr = C.long(len(value))
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
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN) C.SQLRETURN {

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	//value := s.data[s.index][Col_or_Param_Num-1]
	//fmt.Printf("%d %v %T\n", Col_or_Param_Num-1, value, value)

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
	NumericAttributePtr *C.SQLLEN) C.SQLRETURN {

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
	fmt.Printf("SQLRowCount %q %v\n", StatementHandle, RowCountPtr)

	*RowCountPtr = 1

	return C.SQL_SUCCESS
}

//export SQLCancel
func SQLCancel(StatementHandle C.SQLHSTMT) C.SQLRETURN {
	fmt.Printf("SQLCancel %q\n", StatementHandle)

	return C.SQL_SUCCESS

}

//export SQLFreeStmt
func SQLFreeStmt(StatementHandle C.SQLHSTMT, Option C.SQLUSMALLINT) C.SQLRETURN {
	fmt.Printf("SQLFreeStmt %q %q\n", StatementHandle, Option)

	return C.SQL_SUCCESS

}

//export SQLGetInfo
func SQLGetInfo(ConnectionHandle C.SQLHDBC, InfoType C.SQLUSMALLINT, InfoValuePtr C.SQLPOINTER,
	BufferLength C.SQLSMALLINT, StringLengthPtr *C.SQLSMALLINT) C.SQLRETURN {
	fmt.Printf("SQLGetInfo %v %d %v %q %v\n", ConnectionHandle, InfoType, InfoValuePtr, BufferLength, StringLengthPtr)
	return C.SQL_SUCCESS

}

//export SQLDisconnect
func SQLDisconnect(ConnectionHandle C.SQLHDBC) C.SQLRETURN {
	fmt.Printf("SQLGeSQLDisconnecttInfo %v\n", ConnectionHandle)
	return C.SQL_SUCCESS
}

//export SQLSetConnectAttr
func SQLSetConnectAttr(
	ConnectionHandle C.SQLHDBC,
	Attribute C.SQLINTEGER,
	ValuePtr C.SQLPOINTER,
	StringLength C.SQLINTEGER) C.SQLRETURN {

	return C.SQL_SUCCESS
}

// SQLBindCol
// SQLBindParameter
// SQLCancel
// SQLColumnsW
// SQLConnectW
// SQLDescribeColW
// SQLDescribeParam
// SQLDisconnect
// SQLExecute
// SQLFetchScroll
// SQLFreeHandle
// SQLFreeStmt
// SQLGetInfoW
// SQLNumResultCols
// SQLPrepareW
// SQLSetConnectAttr
// SQLSetEnvAttr
// SQLSetStmtAttr
// SQLTablesW
