package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/phobos/client"
	"github.com/marsevilspirit/phobos/example"
)

var (
	deimosAddr = flag.String("deimosAddr", "http://127.0.0.1:4001", "deimosaddress")
	basePath   = flag.String("base", "/phobos/HelloWorld", "prefix path")
)

func main() {
	flag.Parse()

	d := client.NewDeimosDiscovery(*basePath, []string{*deimosAddr})

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
