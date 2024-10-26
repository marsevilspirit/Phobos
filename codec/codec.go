package codec

import (
	"encoding/json"
	"fmt"

	pb "github.com/gogo/protobuf/proto"
	"github.com/vmihailenco/msgpack/v5"
)

type Codec interface {
	Encode(i interface{}) ([]byte, error)
	Decode(data []byte, i interface{}) error
}

type ByteCodec struct{}

func (c ByteCodec) Encode(i interface{}) ([]byte, error) {
	if data, ok := i.([]byte); ok {
		return data, nil
	}

	//%T获取i的类型
	return nil, fmt.Errorf("%T is not a []byte", i)
}

func (c ByteCodec) Decode(data []byte, i interface{}) error {
	i = &data
	return nil
}

type JSONCodec struct{}

func (c JSONCodec) Encode(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

func (c JSONCodec) Decode(data []byte, i interface{}) error {
	return json.Unmarshal(data, i)
}

type ProtobufCodec struct{}

func (c ProtobufCodec) Encode(i interface{}) ([]byte, error) {
	if m, ok := i.(pb.Marshaler); ok {
		return m.Marshal()
	}

	return nil, fmt.Errorf("%T is not a pb.Marshaler", i)
}

func (c ProtobufCodec) Decode(data []byte, i interface{}) error {
	if m, ok := i.(pb.Unmarshaler); ok {
		return m.Unmarshal(data)
	}

	return fmt.Errorf("%T is not a proto.Unmarshaler", i)
}

type MsgpackCodec struct{}

func (c MsgpackCodec) Encode(i interface{}) ([]byte, error) {
	return msgpack.Marshal(i)
}

func (c MsgpackCodec) Decode(data []byte, i interface{}) error {
	return msgpack.Unmarshal(data, i)
}
