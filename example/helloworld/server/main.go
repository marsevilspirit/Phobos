package main

import (
	"flag"

	"github.com/marsevilspirit/phobos/example"
	"github.com/marsevilspirit/phobos/server"
)

var (
	addr = flag.String("addr", "localhost:30000", "server address")
)

func main() {
	flag.Parse()

	s := server.NewServer()
	s.RegisterWithName("HelloWorld", new(example.HelloWorld), "")
	s.Serve("tcp", *addr)
}
