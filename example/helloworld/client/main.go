package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/phobos/client"
	"github.com/marsevilspirit/phobos/example"
)

var (
	addr = flag.String("addr", "localhost:30000", "server address")
)

func main() {
	flag.Parse()

	d := client.NewP2PDiscovery("tcp@"+*addr, "")
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

	log.Print("reply: ", reply.Second)
}
