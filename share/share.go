package share

import (
	"github.com/marsevilspirit/phobos/codec"
	"github.com/marsevilspirit/phobos/protocol"
)

const (
	// used by HTTP
	DefaultRPCPath = "/_phobos_"

	// used by auth
	AuthKey = "__AUTH"
)

var (
	Codecs = map[protocol.SerializeType]codec.Codec{
		protocol.SerializeNone: &codec.ByteCodec{},
		protocol.JSON:          &codec.JSONCodec{},
		protocol.ProtoBuffer:   &codec.ProtobufCodec{},
		protocol.MsgPack:       &codec.MsgpackCodec{},
	}
)

// 对外暴露的注册方法
func RegisterCodec(t protocol.SerializeType, c codec.Codec) {
	Codecs[t] = c
}

// ContextKey is a type for context keys
type ContextKey string

var ReqMetaDataKey = ContextKey("reqMetaData")

var ResMetaDataKey = ContextKey("resMetaData")
