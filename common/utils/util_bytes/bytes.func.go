package util_bytes

import "unsafe"

// ToBytes 字符串转 []byte，避免 []byte(str) 带来的数据复制，转出的数据不可写
func ToBytes(s string) []byte {
	// v1
	return unsafe.Slice(unsafe.StringData(s), len(s))
	// v2
	// return *(*[]byte)(unsafe.Pointer(&s))
}

// ToString []byte 转字符串，避免 string([]byte) 带来的数据复制
func ToString(b []byte) string {
	// v1
	return unsafe.String(unsafe.SliceData(b), len(b))
	// v2
	// return *(*string)(unsafe.Pointer(&b))
}
