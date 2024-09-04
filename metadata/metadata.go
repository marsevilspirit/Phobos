package metadata

import (
	"context"
	"fmt"
	"strings"
)

// metadata 是一个键值对映射的元数据，每个键对应多个值。
type metadata map[string][]string

// NewMetadata 创建一个元数据。
// 传入的参数是一个 map[string]string，函数将其转换为 Metadata 类型。
func NewMetadata(m map[string]string) metadata {
	md := make(metadata, len(m))
	for key, value := range m {
		key := strings.ToLower(key)
		md[key] = append(md[key], value)
	}
	return md
}

// AddMetadata 将 metadata 元数据添加到上下文中。
func AddMetadata(ctx context.Context, md metadata) context.Context {
	return context.WithValue(ctx, "metadata", md)
}

// GetMetadata 从上下文中提取 metadata 元数据。
func GetMetadata(ctx context.Context) metadata {
	if md, ok := ctx.Value("metadata").(metadata); ok {
		return md
	}
	return nil
}

func Pairs(kv ...string) metadata {
	if len(kv)%2 == 1 {
		panic(fmt.Sprintf("metadata: Pairs got the odd number of input pairs for metadata: %d", len(kv)))
	}
	md := make(metadata, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key := strings.ToLower(kv[i])
		md[key] = append(md[key], kv[i+1])
	}
	return md
}

func Join(mds ...metadata) metadata {
	out := metadata{}
	for _, md := range mds {
		for k, v := range md {
			out[k] = append(out[k], v...)
		}
	}
	return out
}
