package codec

import (
	"testing"
)

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
	input := &ProtoArgs{A: 5, B: 10}

	encoded, err := codec.Encode(input)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	var output ProtoArgs
	err = codec.Decode(encoded, &output)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if output.A != 5 || output.B != 10 {
		t.Fatalf("expected A = 5, B = 10 but got A = %d, B = %d", output.A, output.B)
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
