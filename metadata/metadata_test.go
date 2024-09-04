package metadata

import (
	"testing"
	"reflect"
)

func TestJoin(t *testing.T) {
	for _, test := range []struct {
		mds  []metadata
		want metadata
	}{
		{[]metadata{}, metadata{}},
		{[]metadata{Pairs("foo", "bar")}, Pairs("foo", "bar")},
		{[]metadata{Pairs("foo", "bar"), Pairs("foo", "baz")}, Pairs("foo", "bar", "foo", "baz")},
		{[]metadata{Pairs("foo", "bar"), Pairs("foo", "baz"), Pairs("zip", "zap")}, Pairs("foo", "bar", "foo", "baz", "zip", "zap")},
	} {
		md := Join(test.mds...)
		if !reflect.DeepEqual(md, test.want) {
			t.Errorf("context's metadata is %v, want %v", md, test.want)
		}
	}
}
