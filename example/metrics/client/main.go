package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/m_RPC/client"
	"github.com/marsevilspirit/m_RPC/example"
)

var (
	addr = flag.String("addr", "localhost:30000", "server address")
)

func main() {
	flag.Parse()

	d := client.NewP2PDiscovery("tcp@"+*addr, "")
	xclient1 := client.NewXClient("HelloWorld", "Greet", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	xclient2 := client.NewXClient("HelloWorld", "Greet", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	xclient3 := client.NewXClient("HelloWorld", "Greet", client.Failtry, client.RandomSelect, d, client.DefaultOption)

	defer xclient1.Close()
	defer xclient2.Close()
	defer xclient3.Close()

	args := &example.Args{
		First: "budei",
	}

	reply := &example.Reply{}

	for i := 0; i < 10; i++ {

		err := xclient1.Call(context.Background(), args, reply)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}
		log.Print("reply1: ", reply.Second)

		err = xclient2.Call(context.Background(), args, reply)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}
		log.Print("reply2: ", reply.Second)

		err = xclient3.Call(context.Background(), args, reply)
		if err != nil {
			log.Fatalf("failed to call: %v", err)
		}
		log.Print("reply3: ", reply.Second)
	}

	select {}
}
