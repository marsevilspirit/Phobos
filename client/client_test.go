package client

import (
	"context"
	"testing"
	"time"

	"github.com/marsevilspirit/m_RPC/protocol"
	"github.com/marsevilspirit/m_RPC/server"
)

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
}

type Arith int

func (t *Arith) Mul(ctx context.Context, args *Args, reply *Reply) error {
	reply.C = args.A * args.B
	return nil
}

type PBArith int

func (t *PBArith) Mul(ctx context.Context, args *ProtoArgs, reply *ProtoReply) error {
	reply.C = args.A * args.B
	return nil
}

func TestClient_IT(t *testing.T) {
	s := server.Server{}
	s.RegisterWithName("Arith", new(Arith), "")
	s.RegisterWithName("PBArith", new(PBArith), "")
	go s.Serve("tcp", "127.0.0.1:0")
	defer s.Close()
	time.Sleep(500 * time.Millisecond)

	addr := s.Address().String()

	c := &Client{
		option: DefaultOption,
	}

	err := c.Connect("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer c.Close()

	args := &Args{
		A: 10,
		B: 20,
	}

	reply := &Reply{}
	err = c.Call(context.Background(), "Arith", "Mul", args, reply, nil)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if reply.C != 200 {
		t.Fatalf("expect 200 but got %d", reply.C)
	}

	err = c.Call(context.Background(), "Arith", "Add", args, reply, nil)
	if err == nil {
		t.Fatal("expect an error but got nil")
	}

	c.option.SerializeType = protocol.MsgPack
	reply = &Reply{}
	err = c.Call(context.Background(), "Arith", "Mul", args, reply, nil)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if reply.C != 200 {
		t.Fatalf("expect 200 but got %d", reply.C)
	}

	c.option.SerializeType = protocol.ProtoBuffer

	pbArgs := &ProtoArgs{
		A: 10,
		B: 20,
	}
	pbReply := &ProtoReply{}
	err = c.Call(context.Background(), "PBArith", "Mul", pbArgs, pbReply, nil)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if pbReply.C != 200 {
		t.Fatalf("expect 200 but got %d", pbReply.C)
	}
}
