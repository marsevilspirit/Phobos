package util

import (
	"testing"
)

func TestGetFreePort(t *testing.T) {
	for i := 0; i < 1000; i++ {
		port, err := GetFreePort()
		if err != nil {
			t.Error(err)
		}

		if port == 0 {
			t.Error("GetFreePort() return 0")
		}
	}
}
