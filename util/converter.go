package util

// 日后可以使用unsafe pointer来避免内存拷贝

func SliceByteToString(b []byte) string {
	return string(b)
}

func StringToSliceByte(s string) []byte {
	return []byte(s)
}

// CopyMeta copy meta from src to dst
func CopyMeta(src, dst map[string]string) {
	if dst == nil {
		return
	}

	for k, v := range src {
		dst[k] = v
	}
}
