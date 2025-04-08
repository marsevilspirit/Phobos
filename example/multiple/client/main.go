package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/phobos/client"
	"github.com/marsevilspirit/phobos/example"
)

var (
	addr1 = flag.String("addr1", "localhost:30000", "server1 address")
	addr2 = flag.String("addr2", "localhost:30001", "server2 address")
)

func main() {
	flag.Parse()

	d := client.NewMultipleServersDiscovery([]*client.KVPair{{Key: *addr1}, {Key: *addr2}})
	xclient := client.NewXClient("HelloWorld", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	i := 0

	args := &example.Args{
		First: "many",
	}
	reply := &example.Reply{}
	err := xclient.Call(context.Background(), "Greet", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	log.Printf("reply: %v %d\n", reply.Second, i)

	i++
}
