package main

// #cgo LDFLAGS: -lodbcinst
// #include <odbcinst.h>
// #include <stdlib.h>
import "C"
import "unsafe"

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
