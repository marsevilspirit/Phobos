package example

import "context"

type Args struct {
	First string
}

type Reply struct {
	Second string
}

type HelloWorld int

func (t *HelloWorld) Greet(ctx context.Context, args *Args, reply *Reply) error {
	reply.Second = "Hello " + args.First
	return nil
}
