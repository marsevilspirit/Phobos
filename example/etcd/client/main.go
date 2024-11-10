package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/m_RPC/client"
	"github.com/marsevilspirit/m_RPC/example"
)

var (
	etcdAddr = flag.String("etcdAddr", "localhost:2379", "etcdaddress")
	basePath = flag.String("base", "/mrpc_example/HelloWorld", "prefix path")
)

func main() {
	flag.Parse()

	d := client.NewEtcdDiscovery(*basePath, []string{*etcdAddr})

	xclient := client.NewXClient("HelloWorld", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	args := &example.Args{
		First: "budei",
	}

	reply := &example.Reply{}

	err := xclient.Call(context.Background(), "Greet", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	log.Printf("reply: %v", reply.Second)
}
