//go:build !odbcinst

package main

func SQLGetPrivateProfileString(section, entry, defaultValue, filename string) string {
	return defaultValue
}
