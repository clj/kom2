//go:build odbcinst && !windows

package main

// #cgo LDFLAGS: -lodbcinst
import "C"
