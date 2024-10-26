package codec

import (
	"encoding/json"
	"testing"
)

// MockProto 是一个模拟的 proto.Message，用于测试 ProtobufCodec
type MockProto struct {
	Field1 string
}

func (m *MockProto) Reset()         {}
func (m *MockProto) String() string { return "" }
func (m *MockProto) ProtoMessage()  {}
func (m *MockProto) Marshal() ([]byte, error) {
	return json.Marshal(m)
}
func (m *MockProto) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

func TestByteCodec(t *testing.T) {
	codec := ByteCodec{}
	input := []byte("test data")

	encoded, err := codec.Encode(input)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if string(encoded) != string(input) {
		t.Fatalf("expected %s but got %s", input, encoded)
	}

	_, err = codec.Encode("not a byte slice")
	if err == nil {
		t.Fatalf("expected an error but got none")
	}
}

func TestJSONCodec(t *testing.T) {
	codec := JSONCodec{}
	input := map[string]string{"key": "value"}

	encoded, err := codec.Encode(input)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	var output map[string]string
	err = codec.Decode(encoded, &output)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if output["key"] != "value" {
		t.Fatalf("expected %s but got %s", "value", output["key"])
	}
}

func TestProtobufCodec(t *testing.T) {
	codec := ProtobufCodec{}
	input := &MockProto{Field1: "test"}

	encoded, err := codec.Encode(input)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	var output MockProto
	err = codec.Decode(encoded, &output)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if output.Field1 != "test" {
		t.Fatalf("expected %s but got %s", "test", output.Field1)
	}

	_, err = codec.Encode("not a proto.Message")
	if err == nil {
		t.Fatalf("expected an error but got none")
	}
}

func TestMsgpackCodec(t *testing.T) {
	codec := MsgpackCodec{}
	input := map[string]string{"key": "value"}

	encoded, err := codec.Encode(input)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	var output map[string]string
	err = codec.Decode(encoded, &output)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if output["key"] != "value" {
		t.Fatalf("expected %s but got %s", "value", output["key"])
	}
}
