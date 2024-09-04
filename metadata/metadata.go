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

// Append 将一个或多个值追加到指定的元数据键中。
// 如果键不存在，它会创建一个新的键，并将值添加到该键对应的切片中。
// 如果键已经存在，新的值将被追加到现有的切片后面。
func (md metadata) Append(k string, vals ...string) {
	if len(vals) == 0 {
		return
	}
	k = strings.ToLower(k)
	md[k] = append(md[k], vals...)
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

func (md metadata) Get(key string) string {
	if md == nil {
		return ""
	}
	return strings.Join(md[key], ",")
}

type metadataKey struct{}

// AddMetadata 将 metadata 元数据添加到上下文中。
func AddMetadataToContext(ctx context.Context, md metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, md)
}

// GetMetadata 从上下文中提取 metadata 元数据。
func GetMetadataFromContext(ctx context.Context) metadata {
	if md, ok := ctx.Value(metadataKey{}).(metadata); ok {
		return md
	}
	return nil
}
