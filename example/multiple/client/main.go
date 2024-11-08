package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/m_RPC/client"
	"github.com/marsevilspirit/m_RPC/example"
)

var (
	addr1 = flag.String("addr1", "localhost:30000", "server1 address")
	addr2 = flag.String("addr2", "localhost:30001", "server2 address")
)

func main() {
	flag.Parse()

	d := client.NewMultipleServersDiscovery([]*client.KVPair{{Key: *addr1}, {Key: *addr2}})
	xclient := client.NewXClient("HelloWorld", "Greet", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	i := 0

	args := &example.Args{
		First: "many",
	}
	reply := &example.Reply{}
	err := xclient.Call(context.Background(), args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	log.Printf("reply: %v %d\n", reply.Second, i)

	i++
}
