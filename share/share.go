package share

import (
	"github.com/marsevilspirit/m_RPC/codec"
	"github.com/marsevilspirit/m_RPC/protocol"
)

const (
	// used by HTTP
	DefaultRPCPath = "/_mpc_"
)

var (
	Codecs = map[protocol.SerializeType]codec.Codec{
		protocol.SerializeNone: &codec.ByteCodec{},
		protocol.JSON:          &codec.JSONCodec{},
		protocol.ProtoBuffer:   &codec.ProtobufCodec{},
		protocol.MsgPack:       &codec.MsgpackCodec{},
	}
)
