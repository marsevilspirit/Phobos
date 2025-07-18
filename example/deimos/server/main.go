// go run -tags deimos server.go
package main

import (
	"flag"
	"log"
	"time"

	"github.com/marsevilspirit/phobos/example"
	"github.com/marsevilspirit/phobos/server"
	"github.com/marsevilspirit/phobos/serverplugin"
)

var (
	addr       = flag.String("addr", "localhost:30000", "server address")
	deimosAddr = flag.String("deimosAddr", "http://127.0.0.1:4001", "etcd address")
)

func main() {
	flag.Parse()

	s := server.NewServer()
	addRegistryPlugin(s)

	s.RegisterWithName("HelloWorld", new(example.HelloWorld), "")
	s.Serve("tcp", *addr)
}

func addRegistryPlugin(s *server.Server) {
	r := &serverplugin.DeimosRegisterPlugin{
		ServiceAddress: "tcp@" + *addr,
		DeimosServers:  []string{*deimosAddr},
		UpdateInterval: time.Minute,
	}

	err := r.Start()
	if err != nil {
		log.Fatal(err)
	}

	s.Plugins.Add(r)
}
