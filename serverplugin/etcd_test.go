//go:build etcd
// +build etcd

package serverplugin

import (
	"testing"
	"time"

	"github.com/marsevilspirit/phobos/server"
	metrics "github.com/rcrowley/go-metrics"
)

func TestEtcdRegistry(t *testing.T) {
	s := server.NewServer()

	r := &EtcdRegisterPlugin{
		ServiceAddress: "tcp@127.0.0.1:8972",
		EtcdServers:    []string{"127.0.0.1:2379"},
		BasePath:       "/mrpc_test",
		Metrics:        metrics.NewRegistry(),
		UpdateInterval: time.Minute,
	}
	err := r.Start()
	if err != nil {
		t.Fatal(err)
	}
	s.Plugins.Add(r)

	s.RegisterWithName("Arith", new(Arith), "")
	go s.Serve("tcp", "127.0.0.1:8972")
	defer s.Close()

	if len(r.Services) == 0 {
		t.Fatal("failed to register services in etcd")
	}
}
