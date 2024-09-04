package codec

import (
    "google.golang.org/protobuf/proto"
    "errors"
)

// Codec 是用于不同序列化格式的通用接口
type Codec interface {
    Encode(v interface{}) ([]byte, error)
    Decode(data []byte, v interface{}) error
}

// ProtobufCodec 实现了 Codec 接口，用于 Protobuf 编解码
type ProtobufCodec struct{}

// Encode 将数据对象编码为 Protobuf 字节流
func (p *ProtobufCodec) Encode(v interface{}) ([]byte, error) {
    // 检查传入的对象是否实现了 proto.Message 接口
    msg, ok := v.(proto.Message)
    if !ok {
        return nil, errors.New("invalid message type: does not implement proto.Message")
    }
    // 使用 Protobuf 序列化
    return proto.Marshal(msg)
}

// Decode 从 Protobuf 字节流解码为数据对象
func (p *ProtobufCodec) Decode(data []byte, v interface{}) error {
    // 检查传入的对象是否实现了 proto.Message 接口
    msg, ok := v.(proto.Message)
    if !ok {
        return errors.New("invalid message type: does not implement proto.Message")
    }
    // 使用 Protobuf 反序列化
    return proto.Unmarshal(data, msg)
}
