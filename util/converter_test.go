package util

import (
	"bytes"
	"testing"
)

func TestSliceByteToString(t *testing.T) {
	tests := []struct {
		input  []byte
		expect string
	}{
		{input: []byte("hello"), expect: "hello"},
		{input: []byte("world"), expect: "world"},
		{input: []byte(""), expect: ""},
	}

	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			output := SliceByteToString(test.input)
			if output != test.expect {
				t.Errorf("expected %s, got %s", test.expect, output)
			}
		})
	}
}

func TestStringToSliceByte(t *testing.T) {
	tests := []struct {
		input  string
		expect []byte
	}{
		{input: "hello", expect: []byte("hello")},
		{input: "world", expect: []byte("world")},
		{input: "", expect: []byte("")},
		{input: "你好", expect: []byte("你好")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			output := StringToSliceByte(tt.input)
			if !bytes.Equal(output, tt.expect) {
				t.Errorf("expected %s, got %s", tt.expect, output)
			}
		})
	}
}
