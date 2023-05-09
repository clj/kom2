package main

import (
	"fmt"
	"unsafe"
)

import "C"

type SQLSMALLINT = C.short
type SQLRETURN = SQLSMALLINT
type SQLHANDLE = unsafe.Pointer
type SQLHDBC = SQLHANDLE
type SQLCHAR = C.uchar
type SQLSCHAR = C.char
type SQLHSTMT = SQLHANDLE
type SQLPOINTER = unsafe.Pointer
type SQLINTEGER = C.int
type SQLUINTEGER = C.uint
type SQLUSMALLINT = C.ushort
type SQLULEN = SQLUINTEGER
type SQLLEN = SQLINTEGER

/****************************
 * some ret values
 ***************************/
//  #define SQL_NULL_DATA             (-1)
//  #define SQL_DATA_AT_EXEC          (-2)
//  #define SQL_SUCCESS                0
const SQL_SUCCESS SQLRETURN = 0

// #define SQL_SUCCESS_WITH_INFO      1
// #if (ODBCVER >= 0x0300)
// #define SQL_NO_DATA              100
const SQL_NO_DATA SQLRETURN = 100

//  #endif
//  #define SQL_ERROR                 (-1)
//  #define SQL_INVALID_HANDLE        (-2)
//  #define SQL_STILL_EXECUTING        2
//  #define SQL_NEED_DATA             99
//  #define SQL_SUCCEEDED(rc) (((rc)&(~1))==0)

/****************************
 * use these to indicate string termination to some function
 ***************************/
//  #define SQL_NTS                   (-3)
//  #define SQL_NTSL                  (-3L)
const SQL_NTS SQLSMALLINT = -3

//const SQL_NTSL = -3

func toGoString(str *SQLSCHAR, len SQLSMALLINT) string {
	if str == nil {
		return ""
	}
	if len == SQL_NTS {
		return C.GoString(str)
	}
	return C.GoStringN(str, C.int(len))
}

// SQLRETURN SQLConnect(
//
//	SQLHDBC        ConnectionHandle,
//	SQLCHAR *      ServerName,
//	SQLSMALLINT    NameLength1,
//	SQLCHAR *      UserName,
//	SQLSMALLINT    NameLength2,
//	SQLCHAR *      Authentication,
//	SQLSMALLINT    NameLength3);
//
//export SQLConnect
func SQLConnect(ConnectionHandle SQLHDBC,
	ServerName *SQLSCHAR, NameLength1 SQLSMALLINT,
	UserName *SQLSCHAR, NameLength2 SQLSMALLINT,
	Authentication *SQLSCHAR, NameLength3 SQLSMALLINT) SQLRETURN {

	fmt.Printf("%v %d %v %d %v %d\n", ServerName, NameLength1, UserName, NameLength2, Authentication, NameLength3)
	serverName := toGoString(ServerName, NameLength1)
	userName := toGoString(UserName, NameLength2)
	authentication := toGoString(Authentication, NameLength3)

	fmt.Printf("ServerName: %s\n", serverName)
	fmt.Printf("UserName: %s\n", userName)
	fmt.Printf("Authentication: %s\n", authentication)

	return SQL_SUCCESS
}

// SQLRETURN SQLAllocHandle(
//
//	SQLSMALLINT   HandleType,
//	SQLHANDLE     InputHandle,
//	SQLHANDLE *   OutputHandlePtr);
//
//export SQLAllocHandle
func SQLAllocHandle(HandleType SQLSMALLINT, InputHandle SQLHANDLE, OutputHandlePtr *SQLHANDLE) SQLRETURN {
	return SQL_SUCCESS
}

// SQLRETURN SQLPrepare(
//
//	SQLHSTMT      StatementHandle,
//	SQLCHAR *     StatementText,
//	SQLINTEGER    TextLength);
//
//export SQLPrepare
func SQLPrepare(StatementHandle SQLHSTMT, StatementText *SQLSCHAR, TextLength SQLSMALLINT) SQLRETURN {
	statementHandle := toGoString(StatementText, TextLength)

	fmt.Printf("StatementText: %s\n", statementHandle)

	return SQL_SUCCESS
}

// SQLRETURN SQLExecute(
//
//	SQLHSTMT     StatementHandle);
//
//export SQLExecute
func SQLExecute(StatementHandle SQLHSTMT) SQLRETURN {
	fmt.Printf("MOOO")

	return SQL_SUCCESS
}

// SQLRETURN SQLNumResultCols(
//
//	SQLHSTMT        StatementHandle,
//	SQLSMALLINT *   ColumnCountPtr);
//
//export SQLNumResultCols
func SQLNumResultCols(StatementHandle SQLHSTMT, ColumnCountPtr *SQLSMALLINT) SQLRETURN {
	fmt.Printf("MOOO2")

	*ColumnCountPtr = 5

	return SQL_SUCCESS
}

// SQLRETURN SQLFreeHandle(
//
//	SQLSMALLINT   HandleType,
//	SQLHANDLE     Handle);
//
//export SQLFreeHandle
func SQLFreeHandle(HandleType SQLSMALLINT, Handle SQLHANDLE) SQLRETURN {
	return SQL_SUCCESS
}

// SQLRETURN SQLTables(
// 	SQLHSTMT       StatementHandle,
// 	SQLCHAR *      CatalogName,
// 	SQLSMALLINT    NameLength1,
// 	SQLCHAR *      SchemaName,
// 	SQLSMALLINT    NameLength2,
// 	SQLCHAR *      TableName,
// 	SQLSMALLINT    NameLength3,
// 	SQLCHAR *      TableType,
// 	SQLSMALLINT    NameLength4);

//export SQLTables
func SQLTables(StatementHandle SQLHSTMT, CatalogName *SQLSCHAR, NameLength1 SQLSMALLINT, SchemaName *SQLSCHAR, NameLength2 SQLSMALLINT, TableName *SQLSCHAR, NameLength3 SQLSMALLINT, TableType *SQLSCHAR, NameLength4 SQLSMALLINT) SQLRETURN {
	catalogName := toGoString(CatalogName, NameLength1)
	schemaName := toGoString(SchemaName, NameLength1)
	tableName := toGoString(TableName, NameLength1)
	tableType := toGoString(TableType, NameLength1)

	fmt.Printf("SQLTables %q  %q %q  %q\n", catalogName, schemaName, tableName, tableType)

	return SQL_SUCCESS
}

// SQLRETURN SQLSetStmtAttr(
//
//	SQLHSTMT      StatementHandle,
//	SQLINTEGER    Attribute,
//	SQLPOINTER    ValuePtr,
//	SQLINTEGER    StringLength);
//
//export SQLSetStmtAttr
func SQLSetStmtAttr(StatementHandle SQLHSTMT, Attribute SQLINTEGER, ValuePtr SQLPOINTER, StringLength SQLINTEGER) SQLRETURN {
	fmt.Printf("SQLSetStmtAttr %q  %q %q  %q\n", StatementHandle, Attribute, ValuePtr, StringLength)

	return SQL_SUCCESS

}

// SQLRETURN SQLDescribeCol(
//
//	SQLHSTMT       StatementHandle,
//	SQLUSMALLINT   ColumnNumber,
//	SQLCHAR *      ColumnName,
//	SQLSMALLINT    BufferLength,
//	SQLSMALLINT *  NameLengthPtr,
//	SQLSMALLINT *  DataTypePtr,
//	SQLULEN *      ColumnSizePtr,
//	SQLSMALLINT *  DecimalDigitsPtr,
//	SQLSMALLINT *  NullablePtr);
//
//export SQLDescribeCol
func SQLDescribeCol(StatementHandle SQLHSTMT, ColumnNumber SQLUSMALLINT, ColumnName *SQLSCHAR, BufferLength SQLSMALLINT,
	NameLengthPtr *SQLSMALLINT, DataTypePtr *SQLSMALLINT, ColumnSizePtr *SQLULEN,
	DecimalDigitsPtr *SQLSMALLINT, NullablePtr *SQLSMALLINT) SQLRETURN {
	fmt.Printf("SQLDescribeCol %q  %q %v %q %v %v %v %v %v\n", StatementHandle, ColumnNumber, ColumnName, BufferLength,
		NameLengthPtr, DataTypePtr, ColumnSizePtr, DecimalDigitsPtr, NullablePtr)

	return SQL_SUCCESS
}

// SQLRETURN SQLBindCol(
// 	SQLHSTMT       StatementHandle,
// 	SQLUSMALLINT   ColumnNumber,
// 	SQLSMALLINT    TargetType,
// 	SQLPOINTER     TargetValuePtr,
// 	SQLLEN         BufferLength,
// 	SQLLEN *       StrLen_or_IndPtr);

//export SQLBindCol
func SQLBindCol(StatementHandle SQLHSTMT, ColumnNumber SQLUSMALLINT, TargetType SQLSMALLINT,
	TargetValuePtr SQLPOINTER, BufferLength SQLLEN, StrLen_or_IndPtr *SQLLEN) SQLRETURN {

	fmt.Printf("SQLBindCol %q  %q %v %q %v %v\n", StatementHandle, ColumnNumber, TargetType, TargetValuePtr,
		BufferLength, StrLen_or_IndPtr)

	return SQL_SUCCESS
}

// SQLRETURN SQLFetchScroll(
//
//	SQLHSTMT      StatementHandle,
//	SQLSMALLINT   FetchOrientation,
//	SQLLEN        FetchOffset);
//
//export SQLFetchScroll
func SQLFetchScroll(StatementHandle SQLHSTMT, FetchOrientation SQLSMALLINT, FetchOffset SQLLEN) SQLRETURN {
	fmt.Printf("SQLFetchScroll %q  %q %q\n", StatementHandle, FetchOrientation, FetchOffset)

	return SQL_NO_DATA
}

// SQLRETURN SQLCancel(
//
//	SQLHSTMT     StatementHandle);
//
//export SQLCancel
func SQLCancel(StatementHandle SQLHSTMT) SQLRETURN {
	fmt.Printf("SQLCancel %q\n", StatementHandle)

	return SQL_SUCCESS

}

// SQLRETURN SQLFreeStmt(
//
//	SQLHSTMT       StatementHandle,
//	SQLUSMALLINT   Option);
//
//export SQLFreeStmt
func SQLFreeStmt(StatementHandle SQLHSTMT, Option SQLUSMALLINT) SQLRETURN {
	fmt.Printf("SQLFreeStmt %q %q\n", StatementHandle, Option)

	return SQL_SUCCESS

}

// SQLRETURN SQLGetInfo(
//
//	SQLHDBC         ConnectionHandle,
//	SQLUSMALLINT    InfoType,
//	SQLPOINTER      InfoValuePtr,
//	SQLSMALLINT     BufferLength,
//	SQLSMALLINT *   StringLengthPtr);
//
//export SQLGetInfo
func SQLGetInfo(ConnectionHandle SQLHDBC, InfoType SQLUSMALLINT, InfoValuePtr SQLPOINTER,
	BufferLength SQLSMALLINT, StringLengthPtr *SQLSMALLINT) SQLRETURN {
	fmt.Printf("SQLGetInfo %q %q %v %q %v\n", ConnectionHandle, InfoType, InfoValuePtr, BufferLength, StringLengthPtr)
	return SQL_SUCCESS

}

// SQLRETURN SQLDisconnect(
//
//	SQLHDBC     ConnectionHandle);
//
//export SQLDisconnect
func SQLDisconnect(ConnectionHandle SQLHDBC) SQLRETURN {
	fmt.Printf("SQLGetInfo %q\n", ConnectionHandle)
	return SQL_SUCCESS
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
