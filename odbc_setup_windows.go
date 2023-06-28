package main

/*
#include <windows.h>
#include <sql.h>
#include <sqltypes.h>
#include <odbcinst.h>


*/
import "C"

//export SQLDataSources
func SQLDataSources(env C.SQLHENV, dir C.SQLUSMALLINT, srvname *C.SQLCHAR,
	buflen1 C.SQLSMALLINT, lenp1 *C.SQLSMALLINT,
	desc *C.SQLCHAR, buflen2 C.SQLSMALLINT, lenp2 *C.SQLSMALLINT,
) C.SQLRETURN {
	if env == C.SQLHENV(uintptr(C.SQL_NULL_HENV)) {
		return C.SQL_INVALID_HANDLE
	}
	return C.SQL_ERROR
}

//export SQLDataSourcesW
func SQLDataSourcesW(env C.SQLHENV, dir C.SQLUSMALLINT, srvname *C.SQLWCHAR,
	buflen1 C.SQLSMALLINT, lenp1 *C.SQLSMALLINT,
	desc *C.SQLWCHAR, buflen2 C.SQLSMALLINT, lenp2 *C.SQLSMALLINT,
) C.SQLRETURN {
	if env == C.SQLHENV(uintptr(C.SQL_NULL_HENV)) {
		return C.SQL_INVALID_HANDLE
	}
	return C.SQL_ERROR
}

//export SQLDrivers
func SQLDrivers(env C.SQLHENV, dir C.SQLUSMALLINT, drvdesc *C.SQLCHAR,
	descmax C.SQLSMALLINT, desclenp *C.SQLSMALLINT,
	drvattr *C.SQLCHAR, attrmax C.SQLSMALLINT, attrlenp *C.SQLSMALLINT) C.SQLRETURN {
	if env == C.SQLHENV(uintptr(C.SQL_NULL_HENV)) {
		return C.SQL_INVALID_HANDLE
	}
	return C.SQL_ERROR
}

//export SQLDriversW
func SQLDriversW(env C.SQLHENV, dir C.SQLUSMALLINT, drvdesc *C.SQLWCHAR,
	descmax C.SQLSMALLINT, desclenp *C.SQLSMALLINT,
	drvattr *C.SQLWCHAR, attrmax C.SQLSMALLINT, attrlenp *C.SQLSMALLINT) C.SQLRETURN {
	if env == C.SQLHENV(uintptr(C.SQL_NULL_HENV)) {
		return C.SQL_INVALID_HANDLE
	}
	return C.SQL_ERROR
}

//export SQLBrowseConnect
func SQLBrowseConnect(dbc C.SQLHDBC, connin *C.SQLCHAR, conninLen C.SQLSMALLINT,
	connout *C.SQLCHAR, connoutMax C.SQLSMALLINT,
	connoutLen *C.SQLSMALLINT) C.SQLRETURN {
	//SQLRETURN ret;

	//C.hdbc_lock(dbc)
	//defer C.hdbc_unlock(dbc)

	const SQL_NULL_HDBC = 0
	if dbc == C.SQLHDBC(uintptr(SQL_NULL_HDBC)) {
		return C.SQL_INVALID_HANDLE
	}
	//d = (DBC *) dbc;
	//setstatd(d, -1, "not supported", "IM001");
	//return ret;
	return C.SQL_ERROR
}

//export SQLBrowseConnectW
func SQLBrowseConnectW(dbc C.SQLHDBC, connin *C.SQLWCHAR, conninLen C.SQLSMALLINT,
	connout *C.SQLWCHAR, connoutMax C.SQLSMALLINT,
	connoutLen *C.SQLSMALLINT) C.SQLRETURN {
	//SQLRETURN ret;

	//C.hdbc_lock(dbc)
	//defer C.hdbc_unlock(dbc)

	const SQL_NULL_HDBC = 0
	if dbc == C.SQLHDBC(uintptr(SQL_NULL_HDBC)) {
		return C.SQL_INVALID_HANDLE
	}
	//d = (DBC *) dbc;
	//setstatd(d, -1, "not supported", "IM001");
	//return ret;
	return C.SQL_ERROR
}

//export SQLGetInfoW
func SQLGetInfoW(dbc C.SQLHDBC, typ C.SQLUSMALLINT, val C.SQLPOINTER, valMax C.SQLSMALLINT,
	valLen *C.SQLSMALLINT) C.SQLRETURN {
	return C.SQL_ERROR
}

//export ConfigDSN
func ConfigDSN(hwnd C.HWND, request C.WORD, driver C.LPCSTR, attribs C.LPCSTR) C.BOOL {
	return 1
}
