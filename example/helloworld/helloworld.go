package helloworld

import "context"

type HelloWorld struct{}

type HelloWorldArgs struct {
	First string
}

type HelloWorldReply struct {
	Last string
}

func (h *HelloWorld) Helloworld(ctx context.Context, args *HelloWorldArgs, reply *HelloWorldReply) error {
	reply.Last = args.First + " world!"
	return nil
}
