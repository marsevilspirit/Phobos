package codec

import (
	"reflect"
	"testing"
)

func TestByteCodec(t *testing.T) {
	codec := ByteCodec{}

	// 测试 Encode 和 Decode 的配合使用
	t.Run("Encode and Decode", func(t *testing.T) {
		// 原始数据
		original := []byte{1, 2, 3, 4, 5}

		// 编码
		encoded, err := codec.Encode(original)
		if err != nil {
			t.Errorf("Unexpected error during encoding: %v", err)
		}

		// 解码到新的切片
		var decoded []byte
		if err := codec.Decode(encoded, &decoded); err != nil {
			t.Errorf("Unexpected error during decoding: %v", err)
		}

		// 验证解码后的数据是否与原始数据相同
		if !reflect.DeepEqual(original, decoded) {
			t.Errorf("Expected %v, but got %v", original, decoded)
		}
	})

	t.Run("Encode into *[]byte and Decode", func(t *testing.T) {
		// 原始数据
		original := &([]byte{6, 7, 8, 9, 10})

		// 编码
		encoded, err := codec.Encode(original)
		if err != nil {
			t.Errorf("Unexpected error during encoding: %v", err)
		}

		// 解码到新的切片
		var decoded []byte
		if err := codec.Decode(encoded, &decoded); err != nil {
			t.Errorf("Unexpected error during decoding: %v", err)
		}

		// 验证解码后的数据是否与原始数据相同
		if !reflect.DeepEqual(*original, decoded) {
			t.Errorf("Expected %v, but got %v", *original, decoded)
		}
	})

	t.Run("Encode with invalid type", func(t *testing.T) {
		input := 123 // 非 []byte 类型
		_, err := codec.Encode(input)
		if err == nil {
			t.Error("Expected an error but got none")
		}
		expectedErrMsg := "int is not a []byte"
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected error message to be '%s', but got '%s'", expectedErrMsg, err.Error())
		}
	})
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
