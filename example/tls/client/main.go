package main

import (
	"context"
	"crypto/tls"
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

	option := client.DefaultOption

	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	option.TLSConfig = conf

	xclient := client.NewXClient("HelloWorld", client.Failtry, client.RandomSelect, d, option)
	defer xclient.Close()

	args := &example.Args{
		First: "budei",
	}

	reply := &example.Reply{}

	err := xclient.Call(context.Background(), "Greet", args, reply)
	if err != nil {
		panic(err)
	}

	log.Printf("reply: %v", reply.Second)
}
