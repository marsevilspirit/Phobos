package codec

import (
	"testing"
	"google.golang.org/protobuf/proto"
)

// TestProtobufCodec_EncodeDecode 测试 ProtobufCodec 的 Encode 和 Decode 功能
func TestProtobufCodec_EncodeDecode(t *testing.T) {
    // 初始化 Protobuf 编解码器
    protobufCodec := &ProtobufCodec{}

    // 创建一个 Request 消息
    originalRequest := &Request{
        Id:   "123",
        Data: "Test data",
    }

    // 测试编码
    encodedData, err := protobufCodec.Encode(originalRequest)
    if err != nil {
        t.Fatalf("Failed to encode: %v", err)
    }

    if len(encodedData) == 0 {
        t.Fatalf("Encoded data is empty")
    }

    // 测试解码
    decodedRequest := &Request{}
    err = protobufCodec.Decode(encodedData, decodedRequest)
    if err != nil {
        t.Fatalf("Failed to decode: %v", err)
    }

    // 检查解码后的数据是否与原始数据一致
    if !proto.Equal(originalRequest, decodedRequest) {
        t.Errorf("Decoded request does not match the original. Got: %+v, Want: %+v", decodedRequest, originalRequest)
    }
}

// TestProtobufCodec_InvalidMessage 测试当输入非 proto.Message 时的错误处理
func TestProtobufCodec_InvalidMessage(t *testing.T) {
    protobufCodec := &ProtobufCodec{}

    // 测试传递不合法的类型给 Encode
    _, err := protobufCodec.Encode("invalid type")
    if err == nil {
        t.Fatal("Expected error when encoding invalid type, got nil")
    }

    // 测试传递不合法的类型给 Decode
    err = protobufCodec.Decode([]byte{}, "invalid type")
    if err == nil {
        t.Fatal("Expected error when decoding into invalid type, got nil")
    }
}
