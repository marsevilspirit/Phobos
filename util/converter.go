package util

// 日后可以使用unsafe pointer来避免内存拷贝

func SliceByteToString(b []byte) string {
	return string(b)
}

func StringToSliceByte(s string) []byte {
	return []byte(s)
}
