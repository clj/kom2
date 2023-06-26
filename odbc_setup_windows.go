package main

/*

 */
import "C"

/*
#include <windows.h>

// #include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sql.h>
#include <sqltypes.h>
#include <sqlext.h>
*/

//export SQLDataSources
func SQLDataSources(env C.SQLHENV, dir C.SQLUSMALLINT, srvname *C.SQLCHAR,
	buflen1 C.SQLSMALLINT, lenp1 *C.SQLSMALLINT,
	desc C.SQLCHAR, buflen2 C.SQLSMALLINT, lenp2 C.SQLSMALLINT,
) C.SQLRETURN {
	if env == C.SQL_NULL_HENV {
		return C.SQL_INVALID_HANDLE
	}
	return C.SQL_ERROR
}

func SQLDrivers( env C.SQLHENV,  dir C.SQLUSMALLINT,  *drvdesc C.SQLCHAR,
            descmax C.SQLSMALLINT,  *desclenp C.SQLSMALLINT,
            *drvattr C.SQLCHAR,  attrmax C.SQLSMALLINT,  *attrlenp C.SQLSMALLINT) C.SQLRETURN {
    if (env == C.SQL_NULL_HENV) {
        return C.SQL_INVALID_HANDLE;
    }
    return C.SQL_ERROR;
}

// SQLRETURN SQL_API
// SQLBrowseConnect(SQLHDBC dbc, SQLCHAR *connin, SQLSMALLINT conninLen,
//                  SQLCHAR *connout, SQLSMALLINT connoutMax,
//                  SQLSMALLINT *connoutLen)
// {
//     SQLRETURN ret;

//     HDBC_LOCK(dbc);
//     ret = drvunimpldbc(dbc);
//     HDBC_UNLOCK(dbc);
//     return ret;
// }
