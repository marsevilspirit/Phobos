package codec

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/vmihailenco/msgpack/v5"
	pb "google.golang.org/protobuf/proto"
)

type Codec interface {
	Encode(i any) ([]byte, error)
	Decode(data []byte, i any) error
}

type ByteCodec struct{}

func (c ByteCodec) Encode(i any) ([]byte, error) {
	if data, ok := i.([]byte); ok {
		return data, nil
	}

	if data, ok := i.(*[]byte); ok {
		return *data, nil
	}

	//%T获取i的类型
	return nil, fmt.Errorf("%T is not a []byte", i)
}

func (c ByteCodec) Decode(data []byte, i any) error {
	v := reflect.Indirect(reflect.ValueOf(i))
	v.SetBytes(data)
	return nil
}

type JSONCodec struct{}

func (c JSONCodec) Encode(i any) ([]byte, error) {
	return json.Marshal(i)
}

func (c JSONCodec) Decode(data []byte, i any) error {
	return json.Unmarshal(data, i)
}

type ProtobufCodec struct{}

func (c ProtobufCodec) Encode(i any) ([]byte, error) {
	if m, ok := i.(pb.Message); ok {
		return pb.Marshal(m)
	}

	return nil, fmt.Errorf("%T is not a proto.Marshaler", i)
}

func (c ProtobufCodec) Decode(data []byte, i any) error {
	if m, ok := i.(pb.Message); ok {
		return pb.Unmarshal(data, m)
	}

	return fmt.Errorf("%T is not a proto.Unmarshaler", i)
}

type MsgpackCodec struct{}

func (c MsgpackCodec) Encode(i any) ([]byte, error) {
	return msgpack.Marshal(i)
}

func (c MsgpackCodec) Decode(data []byte, i any) error {
	return msgpack.Unmarshal(data, i)
}
