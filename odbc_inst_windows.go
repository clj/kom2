//go:build odbcinst && windows

package main

// #cgo LDFLAGS: -lodbccp32
import "C"
