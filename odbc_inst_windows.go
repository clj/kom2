//go:build odbcinst && windows

package main

// #cgo LDFLAGS: -lodbc32 -lodbccp32
import "C"
