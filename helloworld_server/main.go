package main

import (
	"log"
	"net"

	"github.com/marsevilspirit/m_RPC"
	pb "github.com/marsevilspirit/m_RPC/helloworld"
)

type server struct {	
	pb.UnimplementedGreeterServer
}

func main() {
	lis, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	s := mrpc.NewServer()
	log.Printf("Server listening on %v", lis.Addr())
	pb.RegisterGreeterServer(s, &server{})	
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
