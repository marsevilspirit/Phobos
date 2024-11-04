// go run -tags etcd server.go
package main

import (
	"flag"
	"log"
	"time"

	"github.com/marsevilspirit/m_RPC/example"
	"github.com/marsevilspirit/m_RPC/server"
	"github.com/marsevilspirit/m_RPC/serverplugin"
	metrics "github.com/rcrowley/go-metrics"
)

var (
	addr     = flag.String("addr", "localhost:30000", "server address")
	etcdAddr = flag.String("etcdAddr", "localhost:2379", "etcd address")
	basePath = flag.String("base", "/mrpc_example", "prefix path")
)

func main() {
	flag.Parse()

	s := server.NewServer(nil)
	addRegistryPlugin(s)

	s.RegisterWithName("HelloWorld", new(example.HelloWorld), "")
	s.Serve("tcp", *addr)
}

func addRegistryPlugin(s *server.Server) {
	r := &serverplugin.EtcdRegisterPlugin{
		ServiceAddress: "tcp@" + *addr,
		EtcdServers:    []string{*etcdAddr},
		BasePath:       *basePath,
		Metrics:        metrics.NewRegistry(),
		UpdateInterval: time.Minute,
	}

	err := r.Start()
	if err != nil {
		log.Fatal(err)
	}

	s.Plugins.Add(r)
}
