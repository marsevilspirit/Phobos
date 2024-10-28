package main

import (
	"github.com/marsevilspirit/m_RPC/example/helloworld"
	"github.com/marsevilspirit/m_RPC/server"
)

func main() {
	server := server.Server{}
	server.Register(new(helloworld.HelloWorld))
	server.Serve("tcp", "127.0.0.1:50000")
	defer server.Close()
}
