package metadata

import (
	"context"
	"reflect"
	"testing"
)

func TestNewMetadata(t *testing.T) {
	input := map[string]string{
		"foo": "bar",
		"zip": "zap",
	}

	md := NewMetadata(input)

	for k, v := range input {
		if md.Get(k) != v {
			t.Errorf("metadata.Get(%q) = %q, want %q", k, md.Get(k), v)
		}
	}
}

func TestAppend(t *testing.T) {
	md := NewMetadata(map[string]string{
		"foo": "bar",
	})

	md.Append("zip", "zap")

	if md.Get("zip") != "zap" {
		t.Errorf("metadata.Get(%q) = %q, want %q", "zip", md.Get("zip"), "zap")
	}
}

func TestPairs(t *testing.T) {
	md := Pairs("foo", "bar", "zip", "zap")

	if md.Get("foo") != "bar" {
		t.Errorf("metadata.Get(%q) = %q, want %q", "foo", md.Get("foo"), "bar")
	}

	if md.Get("zip") != "zap" {
		t.Errorf("metadata.Get(%q) = %q, want %q", "zip", md.Get("zip"), "zap")
	}
}

func TestAddMetadataToContext(t *testing.T) {
	ctx := context.Background()
	md := Pairs("foo", "bar")
	ctx = AddMetadataToContext(ctx, md)

	if !reflect.DeepEqual(md, GetMetadataFromContext(ctx)) {
		t.Errorf("context's metadata is %v, want %v", GetMetadataFromContext(ctx), md)
	}
}

func TestGetMetadataFromContext(t *testing.T) {
	ctx := context.Background()
	md := Pairs("zip", "zap")
	ctx = AddMetadataToContext(ctx, md)

	if !reflect.DeepEqual(md, GetMetadataFromContext(ctx)) {
		t.Errorf("context's metadata is %v, want %v", GetMetadataFromContext(ctx), md)
	}
}
