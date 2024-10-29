package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/marsevilspirit/m_RPC/breaker"
	"github.com/marsevilspirit/m_RPC/log"
	"github.com/marsevilspirit/m_RPC/protocol"
	"github.com/marsevilspirit/m_RPC/share"
	"github.com/marsevilspirit/m_RPC/util"
)

type Breaker interface {
	Execute(func() (interface{}, error)) (interface{}, error)
}

var defaultBreakerSettings = breaker.Settings{
	Name:        "defaultBreakerSettings",
	MaxRequests: 5,
	Interval:    10 * time.Second,
	Timeout:     30 * time.Second,
}

var defaultBreaker Breaker = breaker.NewBreaker(defaultBreakerSettings)

var (
	ErrShutdown        = errors.New("connection is shutdown")
	ErrUnspportedCodec = errors.New("codec is unsupported")
)

const (
	ReaderBuffsize = 16 * 1024
	WriterBuffsize = 16 * 1024
)

type Call struct {
	ServicePath   string
	ServiceMethod string
	Metadata      map[string]string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

func (call *Call) done() {
	select {
	case call.Done <- call:
		// ok
	default:
		log.Debug("rpc: discarding Call reply due to insufficient Done chan capacity")
	}
}

// for context key
type seqKey struct{}

type Client struct {
	TLSConfig *tls.Config

	Conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer

	SerializeType protocol.SerializeType
	CompressType  protocol.CompressType

	mu       sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // closing 是用户主动关闭的
	shutdown bool // shutdown 是error发生时调用的

	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration

	UseBreaker bool
}

var _ io.Closer = (*Client)(nil)

func (client *Client) Close() error {
	client.mu.Lock()

	for seq, call := range client.pending {
		delete(client.pending, seq)
		if call != nil {
			call.Error = ErrShutdown
			call.done()
		}
	}

	if client.closing || client.shutdown {
		client.mu.Unlock()
		return ErrShutdown
	}

	client.closing = true
	client.mu.Unlock()
	return client.Conn.Close()
}

func (client *Client) Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	if client.UseBreaker {
		_, err := defaultBreaker.Execute(func() (interface{}, error) {
			return nil, client.call(ctx, servicePath, serviceMethod, args, reply)
		})
		return err
	} else {
		return client.call(ctx, servicePath, serviceMethod, args, reply)
	}
}
func (client *Client) call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	seq := new(uint64)
	ctx = context.WithValue(ctx, seqKey{}, seq)
	Done := client.Go(ctx, servicePath, serviceMethod, args, reply, make(chan *Call, 1)).Done

	var err error
	select {
	case <-ctx.Done():
		client.mu.Lock()
		call := client.pending[*seq]
		delete(client.pending, *seq)
		client.mu.Unlock()
		if call != nil {
			call.Error = ctx.Err()
			call.done()
		}

		return ctx.Err()
	case call := <-Done:
		err = call.Error
	}

	return err
}

func (client *Client) Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call {
	call := &Call{
		ServicePath:   servicePath,
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}

	if call.Done == nil {
		call.Done = make(chan *Call, 10)
	} else if cap(call.Done) == 0 {
		log.Panic("rpc: done channel is unbuffered")
	}

	call.Done = done
	client.send(ctx, call)
	return call
}

func (client *Client) send(ctx context.Context, call *Call) {
	client.mu.Lock()
	if client.shutdown || client.closing {
		call.Error = ErrShutdown
		client.mu.Unlock()
		call.done()
		return
	}

	codec := share.Codecs[client.SerializeType]
	if codec == nil {
		call.Error = ErrUnspportedCodec
		client.mu.Unlock()
		call.done()
		return
	}

	if client.pending == nil {
		client.pending = make(map[uint64]*Call)
	}

	seq := client.seq
	client.seq++
	client.pending[seq] = call
	client.mu.Unlock()

	if cseq, ok := ctx.Value(seqKey{}).(*uint64); ok {
		*cseq = seq
	}

	req := protocol.NewMessage()
	req.SetMessageType(protocol.Request)
	req.SetSeq(seq)
	req.SetSerializeType(client.SerializeType)
	req.Metadata[protocol.ServicePath] = call.ServicePath
	req.Metadata[protocol.ServiceMethod] = call.ServiceMethod

	data, err := codec.Encode(call.Args)
	if err != nil {
		log.Error("encode failed: ", err)
		call.Error = err
		call.done()
		return
	}

	if len(data) > 1024 && client.CompressType == protocol.Gzip {
		data, err = util.Zip(data)
		if err != nil {
			call.Error = err
			call.done()
			return
		}

		req.SetCompressType(client.CompressType)
	}

	req.Payload = data

	_, err = req.WriteTo(client.w)
	if err == nil {
		err = client.w.Flush()
	}
	if err != nil {
		client.mu.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mu.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
	}

	if req.IsOneway() {
		client.mu.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mu.Unlock()
		if call != nil {
			call.done()
		}
	}

}

func (client *Client) receive() {
	var err error
	var res *protocol.Message

	for err == nil {
		res, err = protocol.Read(client.r)

		if err != nil {
			break
		}

		seq := res.Seq()
		client.mu.Lock()
		call := client.pending[seq]
		delete(client.pending, seq)
		client.mu.Unlock()

		switch {
		case call == nil:

		case res.MessageStatusType() == protocol.Error:
			call.Error = errors.New(res.Metadata[protocol.ServiceError])
			call.done()
		default:
			data := res.Payload
			if res.CompressType() == protocol.Gzip {
				data, err = util.Unzip(data)
				if err != nil {
					call.Error = errors.New("unzip payload: " + err.Error())
				}
			}

			codec := share.Codecs[res.SerializeType()]
			if codec == nil {
				call.Error = ErrUnspportedCodec
			} else {
				err = codec.Decode(data, call.Reply)
				if err != nil {
					call.Error = err
				}
			}

			call.done()
		}
	}

	client.mu.Lock()
	client.shutdown = true
	closing := client.closing
	if err == io.EOF {
		if closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}

	for _, call := range client.pending {
		call.Error = err
		call.done()
	}

	client.mu.Unlock()
	if err != io.EOF && !closing {
		log.Error("mrpc: client protocol error:", err)
	}
}
