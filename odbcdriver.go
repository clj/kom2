package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/cgo"
	"unsafe"
)

// #cgo LDFLAGS: -lodbcinst
// #include <stdlib.h>
// #include <string.h>
// #include <sqltypes.h>
// #include <sql.h>
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

//export SQLConnect
func SQLConnect(ConnectionHandle C.SQLHDBC,
	ServerName *C.SQLCHAR, NameLength1 C.SQLSMALLINT,
	UserName *C.SQLCHAR, NameLength2 C.SQLSMALLINT,
	Authentication *C.SQLCHAR, NameLength3 C.SQLSMALLINT) C.SQLRETURN {

	fmt.Printf("%v %d %v %d %v %d\n", ServerName, NameLength1, UserName, NameLength2, Authentication, NameLength3)

	connHandle := cgo.Handle(ConnectionHandle).Value().(*connectionHandle)

	// Todo: also pull from the DSN/serverName
	connHandle.inventreeConfig.server = SQLGetPrivateProfileString("kom2", "server", "", ".odbc.ini")
	connHandle.inventreeConfig.apiToken = SQLGetPrivateProfileString("kom2", "password", "", ".odbc.ini")

	// if userName := toGoString(UserName, NameLength2); username != "" {
	// 	connHandle.inventreeConfig.user = userName
	// }
	if authentication := toGoString(Authentication, NameLength3); authentication != "" {
		connHandle.inventreeConfig.apiToken = authentication
	}

	if connHandle.inventreeConfig.server == "" {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "No server specified"})
	}
	if connHandle.inventreeConfig.apiToken == "" {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "No API token (Password) specified"})
	}

	if err := connHandle.updateCategoryMapping(); err != nil {
		return SetAndReturnError(connHandle, &DriverError{SqlState: "08001", Message: "Error updating category list", Err: err})

	}

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

type statementHandle struct {
	errorInfo

	conn  *connectionHandle
	data  [][]any
	index int
}

func (s *statementHandle) init(connHandle *connectionHandle) {
	s.conn = connHandle
	s.index = -1
}

func copyGoStringToCString(dst *C.uchar, src string, length int) {
	cSrc := C.CString(src)
	defer C.free(unsafe.Pointer(cSrc))
	C.strncpy((*C.char)(unsafe.Pointer(dst)), cSrc, C.ulong(length))
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
	fmt.Printf("MOOO")

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)
	s.index = -1
	s.data = [][]any{
		{"a1", "b1", "c1", "d1", "e1"},
		{"a2", "b2", "c2", "d2", "e2"},
		{"a3", "b3", "c3", "d3", "e3"},
		{"a4", "b4", "c4", "d4", "e4"},
		{"a5", "b5", "c5", "d5", "e5"},
		{"a6", "b6", "c6", "d6", "e6"},
	}

	return C.SQL_SUCCESS
}

//export SQLNumResultCols
func SQLNumResultCols(StatementHandle C.SQLHSTMT, ColumnCountPtr *C.SQLSMALLINT) C.SQLRETURN {
	fmt.Printf("MOOO2")

	*ColumnCountPtr = 5

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

	categories := s.conn.categoryMapping
	for name, _ := range categories {
		s.data = append(s.data, []any{nil, nil, name, "TABLE"})
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

	return C.SQL_SUCCESS
}

//export SQLBindCol
func SQLBindCol(StatementHandle C.SQLHSTMT, ColumnNumber C.SQLUSMALLINT, TargetType C.SQLSMALLINT,
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN) C.SQLRETURN {

	fmt.Printf("SQLBindCol %q  %q %v %q %v %v\n", StatementHandle, ColumnNumber, TargetType, TargetValuePtr,
		BufferLength, StrLen_or_IndPtr)

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

	return C.SQL_SUCCESS
}

//export SQLGetData
func SQLGetData(StatementHandle C.SQLHSTMT, Col_or_Param_Num C.SQLUSMALLINT, TargetType C.SQLSMALLINT,
	TargetValuePtr C.SQLPOINTER, BufferLength C.SQLLEN, StrLen_or_IndPtr *C.SQLLEN) C.SQLRETURN {

	s := cgo.Handle(StatementHandle).Value().(*statementHandle)

	switch value := s.data[s.index][Col_or_Param_Num-1].(type) {
	case string:
		dst := (*C.char)(TargetValuePtr)
		src := C.CString(value)
		defer C.free(unsafe.Pointer(src))
		C.strncpy(dst, src, C.ulong(BufferLength))
	case nil:
		StrLen_or_IndPtr = nil
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
	fmt.Printf("SQLGetInfo %q %q %v %q %v\n", ConnectionHandle, InfoType, InfoValuePtr, BufferLength, StringLengthPtr)
	return C.SQL_SUCCESS

}

//export SQLDisconnect
func SQLDisconnect(ConnectionHandle C.SQLHDBC) C.SQLRETURN {
	fmt.Printf("SQLGetInfo %q\n", ConnectionHandle)
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
