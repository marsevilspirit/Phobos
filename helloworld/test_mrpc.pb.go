package helloworld

import (
	context "context"
	errors "errors"
	mrpc "github.com/marsevilspirit/m_RPC"
)

type GreeterServer interface {
	// Sends a greeting
	SayHello(context.Context, *HelloRequest) (*HelloReply, error)
	SayHelloAgain(context.Context, *HelloRequest) (*HelloReply, error)
	mustEmbedUnimplementedGreeterServer()
}

type UnimplementedGreeterServer struct{}

func (UnimplementedGreeterServer) SayHello(context.Context, *HelloRequest) (*HelloReply, error) {
	return nil, errors.New("SayHello unimplemented")
}
func (UnimplementedGreeterServer) SayHelloAgain(context.Context, *HelloRequest) (*HelloReply, error) {
	return nil, errors.New("SayHelloAgain unimplemented")
}
func (UnimplementedGreeterServer) mustEmbedUnimplementedGreeterServer() {}
func (UnimplementedGreeterServer) testEmbeddedByValue()                 {}

func RegisterGreeterServer(s mrpc.ServiceRegistrar, srv GreeterServer) {
	// If the following call pancis, it indicates UnimplementedGreeterServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	
	s.RegisterService(&Greeter_ServiceDesc, srv)
}

func _Greeter_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(HelloRequest)
	if err := dec(in); err != nil {
		return nil, err
	}

	return srv.(GreeterServer).SayHello(ctx, in)
}

func _Greeter_SayHelloAgain_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(HelloRequest)
	if err := dec(in); err != nil {
		return nil, err
	}

	return srv.(GreeterServer).SayHelloAgain(ctx, in)
}

// Greeter_ServiceDesc is the mrpc.ServiceDesc for Greeter service.
// It's only intended for direct use with mrpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Greeter_ServiceDesc = mrpc.ServiceDesc{
	ServiceName: "helloworld.Greeter",
	HandlerType: (*GreeterServer)(nil),
	Methods: []mrpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Greeter_SayHello_Handler,
		},
		{
			MethodName: "SayHelloAgain",
			Handler:    _Greeter_SayHelloAgain_Handler,
		},
	},
	Metadata: "helloworld/helloworld.proto",
}

