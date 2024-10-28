package main

import (
	"context"
	"fmt"

	"github.com/marsevilspirit/m_RPC/client"
	"github.com/marsevilspirit/m_RPC/example/helloworld"
	"github.com/marsevilspirit/m_RPC/log"
	"github.com/marsevilspirit/m_RPC/protocol"
)

func main() {
	client := client.Client{
		SerializeType: protocol.JSON,
		CompressType:  protocol.Gzip,
	}

	err := client.Connect("tcp", "127.0.0.1:50000")
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	args := &helloworld.HelloWorldArgs{
		First: "hello",
	}

	reply := &helloworld.HelloWorldReply{}

	err = client.Call(context.Background(), "HelloWorld", "Helloworld", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	fmt.Println("reply1: ", reply.Last)

	err = client.Call(context.Background(), "HelloWorld", "Helloworld", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	fmt.Println("reply2: ", reply.Last)
}
