//go:build windows

package main

/*
#include <windows.h>
#include <sql.h>
#include <sqlext.h>
#include <odbcinst.h>
#include <stdlib.h>
#cgo LDFLAGS: -lodbc32 -lodbccp32
*/
import "C"

import (
	"flag"
	"fmt"
	"io"
	"os"
	"unsafe"
)

/*
BOOL SQLInstallDriverEx(
     LPCSTR    lpszDriver,
     LPCSTR    lpszPathIn,
     LPSTR     lpszPathOut,
     WORD      cbPathOutMax,
     WORD *    pcbPathOut,
     WORD      fRequest,
     LPDWORD   lpdwUsageCount);


BOOL SQLRemoveDriver(
     LPCSTR   lpszDriver,
     BOOL     fRemoveDSN,
     LPDWORD  lpdwUsageCount);

BOOL SQLConfigDataSource(
     HWND     hwndParent,
     WORD     fRequest,
     LPCSTR   lpszDriver,
     LPCSTR   lpszAttributes);

*/

const subcommandMsg = "expected 'install' or 'uninstall' subcommands"

type SQLInstallError struct {
	Code    int
	Message string
}

func (e *SQLInstallError) Error() string {
	return fmt.Sprintf("code %d: %v", e.Code, e.Message)
}

type SQLInstallErrors struct {
	Errors []SQLInstallError
}

func (e *SQLInstallErrors) Error() string {
	str := ""
	for _, m := range e.Errors {
		str += m.Error() + "\n"
	}

	return str
}

func getSQLInstallerError() *SQLInstallErrors {
	errorMsg := C.malloc(C.size_t(C.SQL_MAX_MESSAGE_LENGTH))
	defer C.free(errorMsg)

	var errs SQLInstallErrors

	for i := 0; ; i++ {
		var errorCode C.DWORD
		var errLen C.WORD
		rc := C.SQLInstallerError(C.ushort(i), &errorCode, C.LPSTR(errorMsg), C.SQL_MAX_MESSAGE_LENGTH-1, &errLen)

		if rc == C.SQL_NO_DATA {
			break
		}

		var err SQLInstallError
		err.Code = int(errorCode)
		err.Message = C.GoStringN((*C.char)(errorMsg), C.int(errLen))

		errs.Errors = append(errs.Errors, err)
	}

	return &errs
}

func driverString(driverName, dll, path string) string {
	if path != "" {
		dll = path + "\\" + dll
	}

	return driverName + "\000Driver=" + dll + "\000Setup=" + dll + "\000ConnectFunctions=YYN\000"
}

func getInstallPath(driverName, dll string) (string, error) {
	const maxPath = 500
	var pathLen C.WORD
	var usageCount C.DWORD

	path := C.malloc(C.size_t(maxPath))
	defer C.free(path)

	if C.SQLInstallDriverEx(C.CString(driverString(driverName, dll, "")), nil, C.LPSTR(path), maxPath, &pathLen, C.ODBC_INSTALL_INQUIRY, &usageCount) != 1 {
		return "", getSQLInstallerError()
	}

	return C.GoString((*C.char)(path)), nil
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func deleteFile(path string) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	C.DeleteFile(cPath)
}

func install(driverName, dll, installPath string) error {
	const maxPath = 500
	var pathLen C.WORD
	var usageCount C.DWORD

	dst := installPath + "\\" + dll
	if _, err := copyFile(dll, dst); err != nil {
		return err
	}

	path := C.malloc(C.size_t(maxPath))
	defer C.free(path)

	if C.SQLInstallDriverEx(C.CString(driverString(driverName, dll, installPath)), C.CString(installPath), C.LPSTR(path), maxPath, &pathLen, C.ODBC_INSTALL_COMPLETE, &usageCount) != 1 {
		return getSQLInstallerError()
	}

	return nil
}

type configDataSourceAction int

const (
	addDsn    configDataSourceAction = C.ODBC_ADD_SYS_DSN
	removeDsn                        = C.ODBC_REMOVE_SYS_DSN
)

func configDataSource(driverName, dsnName string, action configDataSourceAction) error {
	attr := "DSN=" + dsnName + "\000Database=\000"

	if C.SQLConfigDataSource(nil, C.ushort(action), C.CString(driverName), C.CString(attr)) != 1 {
		return getSQLInstallerError()
	}

	return nil
}

func main() {
	var dll, driverName, dsnName string

	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	installCmd.StringVar(&dll, "dll", "kom2.dll", "name of the dll")
	installCmd.StringVar(&driverName, "driver", "kom2", "driver name")
	installCmd.StringVar(&dsnName, "dsnName", "kom2", "dsn name")

	uninstallCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	uninstallCmd.StringVar(&dll, "dll", "kom2.dll", "name of the dll")
	uninstallCmd.StringVar(&driverName, "driver", "kom2", "driver name")
	uninstallCmd.StringVar(&dsnName, "dsnName", "kom2", "dsn name")

	if len(os.Args) < 2 {
		fmt.Println(subcommandMsg)
		os.Exit(1)
	}

	switch os.Args[1] {

	case "install":
		var err error
		var path string
		if path, err = getInstallPath(dll); err != nil {
			panic(err)
		}
		installCmd.Parse(os.Args[2:])
		if err = install(driverName, dll, path); err != nil {
			panic(err)
		}
		configDataSource(removeDsn)
		if err = configDataSource(driverName, dsnName, addDsn); err != nil {
			panic(err)
		}
	case "uninstall":
		uninstallCmd.Parse(os.Args[2:])
		var err error
		var path string
		if path, err = getInstallPath(driverName, dll); err != nil {
			panic(err)
		}
		configDataSource(removeDsn)
		driver := driverString(driverName, dll, path)
		var count C.ulong = 1
		for count > 0 {
			if C.SQLRemoveDriver(C.CString(driver), 1, &count) != 1 {
				panic(getSQLInstallerError())
			}
		}

		deleteFile(path)
	default:
		fmt.Println(subcommandMsg)
		os.Exit(1)
	}
}
