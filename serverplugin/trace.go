package serverplugin

import (
	"context"
	"net"
	"reflect"
	"runtime"

	"github.com/marsevilspirit/phobos/protocol"
	"golang.org/x/net/trace"
)

type TracePlugin struct {
}

func (p *TracePlugin) Register(name string, rcvr any, metadata string) error {
	tr := trace.New("phobos.Server", "Register")
	defer tr.Finish()
	tr.LazyPrintf("Registering %s: %T", name, rcvr)
	return nil
}

func (p *TracePlugin) RegisterFunction(name string, fn any, metadata string) error {
	tr := trace.New("phobos.Server", "RegisterFunction")
	defer tr.Finish()
	tr.LazyPrintf("Registering %s: %T", name, fn)
	return nil
}

func (p *TracePlugin) GetFunctionName(i any) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (p *TracePlugin) PostConnAccept(conn net.Conn) (net.Conn, bool) {
	tr := trace.New("phobos.Server", "Accept")
	defer tr.Finish()
	tr.LazyPrintf("Accepting connection from %s", conn.RemoteAddr().String())
	return conn, true
}

func (p *TracePlugin) PostReadRequest(ctx context.Context, r *protocol.Message, e error) error {
	tr := trace.New("phobos.Server", "ReadRequest")
	defer tr.Finish()
	tr.LazyPrintf("read request %s.%s, seq: %d", r.ServicePath, r.ServiceMethod, r.Seq())
	return nil
}

func (p *TracePlugin) PostWriteResponse(ctx context.Context, req *protocol.Message, res *protocol.Message, err error) error {
	tr := trace.New("phobos.Server", "WriteResponse")
	defer tr.Finish()
	if err == nil {
		tr.LazyPrintf("succeed to call %s.%s, seq: %d", req.ServicePath, req.ServiceMethod, req.Seq())
	} else {
		tr.LazyPrintf("failed to call %s.%s, seq: %d : %v", req.Seq, req.ServicePath, req.ServiceMethod, req.Seq(), err)
		tr.SetError()
	}
	return nil
}
