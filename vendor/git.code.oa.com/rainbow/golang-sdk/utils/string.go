package utils

import "unsafe"

// StringToBytes  zero copy string to bytes
func StringToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// BytesToString zero copy string to bytes
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
