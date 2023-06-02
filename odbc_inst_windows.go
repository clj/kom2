//go:build odbcinst && windows

package main

// #cgo LDFLAGS: -lodbc32
import "C"
