package client

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/marsevilspirit/m_RPC/breaker"
	"github.com/marsevilspirit/m_RPC/log"
	"github.com/marsevilspirit/m_RPC/protocol"
	"github.com/marsevilspirit/m_RPC/share"
	"github.com/marsevilspirit/m_RPC/util"
)

const (
	GatewayVersion           = "MRPC-Gateway-Version"
	GatewayMessageType       = "MRPC-Gateway-MessageType"
	GatewayHeartbeat         = "MRPC-Gateway-Heartbeat"
	GatewayOneway            = "MRPC-Gateway-Oneway"
	GatewayMessageStatusType = "MRPC-Gateway-MessageStatusType"
	GatewaySerializeType     = "MRPC-Gateway-SerializeType"
	GatewayMessageID         = "MRPC-Gateway-MessageID"
	GatewayServicePath       = "MRPC-Gateway-ServicePath"
	GatewayServiceMethod     = "MRPC-Gateway-ServiceMethod"
	GatewayMeta              = "MRPC-Gateway-Meta"
	GatewayErrorMessage      = "MRPC-Gateway-ErrorMessage"
)

// ServiceError is the error interface for service error
type ServiceError string

func (e ServiceError) Error() string {
	return string(e)
}

var DefaultOption = Option{
	Retries:        3,
	RPCPath:        share.DefaultRPCPath,
	ConnectTimeout: 10 * time.Second,
	Breaker:        defaultBreaker,
	SerializeType:  protocol.MsgPack,
	CompressType:   protocol.None,
}

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

	Metadata    map[string]string
	ResMetadata map[string]string

	Args  interface{}
	Reply interface{}
	Error error
	Done  chan *Call
	IsRaw bool
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

type RPCClient interface {
	Connect(network, address string) error
	Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call
	Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error
	SendRaw(ctx context.Context, r *protocol.Message) (map[string]string, []byte, error)
	Close() error

	RegisterServerMessageChan(ch chan<- *protocol.Message)
	UnregisterServerMessageChan()

	IsClosing() bool
	IsShutdown() bool
}

type Client struct {
	option Option

	Conn net.Conn
	r    *bufio.Reader
	// w    *bufio.Writer

	mu       sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // closing 是用户主动关闭的
	shutdown bool // shutdown 是error发生时调用的

	Plugins PluginContainer

	ServerMessageChan chan<- *protocol.Message
}

func NewClient(options Option) *Client {
	client := &Client{
		option:  options,
		pending: make(map[uint64]*Call),
	}

	if client.option.ConnectTimeout == 0 {
		client.option.ConnectTimeout = DefaultOption.ConnectTimeout
	}

	if client.option.ReadTimeout == 0 {
		client.option.ReadTimeout = DefaultOption.ReadTimeout
	}

	if client.option.WriteTimeout == 0 {
		client.option.WriteTimeout = DefaultOption.WriteTimeout
	}

	if client.option.Breaker == nil {
		client.option.Breaker = DefaultOption.Breaker
	}

	if client.option.SerializeType == 0 {
		client.option.SerializeType = DefaultOption.SerializeType
	}

	if client.option.CompressType == 0 {
		client.option.CompressType = DefaultOption.CompressType
	}

	if client.option.HeartbeatInterval == 0 {
		client.option.HeartbeatInterval = 3 * time.Second
	}

	return client
}

type Option struct {
	Retries int

	TLSConfig *tls.Config
	RPCPath   string

	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration

	Breaker Breaker

	SerializeType protocol.SerializeType
	CompressType  protocol.CompressType

	Heartbeat bool

	HeartbeatInterval time.Duration
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

func (client *Client) Call(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}) error {
	if client.option.Breaker != nil {
		_, err := client.option.Breaker.Execute(func() (interface{}, error) {
			return nil, client.call(ctx, servicePath, serviceMethod, args, reply)
		})
		return err
	} else {
		return client.call(ctx, servicePath, serviceMethod, args, reply)
	}
}
func (client *Client) call(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}) error {
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
		meta := ctx.Value(share.ResMetaDataKey)
		if meta != nil && len(call.ResMetadata) > 0 {
			resMeta := meta.(map[string]string)
			for k, v := range call.ResMetadata {
				resMeta[k] = v
			}
		}
	}

	return err
}

func (client *Client) SendRaw(ctx context.Context, r *protocol.Message) (map[string]string, []byte, error) {
	ctx = context.WithValue(ctx, seqKey{}, r.Seq())

	call := new(Call)
	call.IsRaw = true
	call.ServicePath = r.ServicePath
	call.ServiceMethod = r.ServiceMethod
	meta := ctx.Value(share.ReqMetaDataKey)
	if meta != nil {
		call.Metadata = meta.(map[string]string)
	}
	done := make(chan *Call, 10)
	call.Done = done

	seq := r.Seq()
	client.mu.Lock()
	if client.pending == nil {
		client.pending = make(map[uint64]*Call)
	}
	client.pending[seq] = call
	client.mu.Unlock()

	data := r.Encode()
	_, err := client.Conn.Write(data)
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
	if r.IsOneway() {
		client.mu.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mu.Unlock()
		if call != nil {
			call.done()
		}
	}

	var m map[string]string
	var payload []byte

	select {
	case <-ctx.Done():
		client.mu.Lock()
		call := client.pending[seq]
		delete(client.pending, seq)
		client.mu.Unlock()
		if call != nil {
			call.Error = ctx.Err()
			call.done()
		}

		return nil, nil, ctx.Err()
	case call := <-done:
		err = call.Error
		m = call.ResMetadata
		if call.Reply != nil {
			payload = call.Reply.([]byte)
		}
	}

	return m, payload, err
}

func convertRes2Raw(res *protocol.Message) (map[string]string, []byte, error) {
	m := make(map[string]string)
	m[GatewayVersion] = strconv.Itoa(int(res.Version()))

	if res.IsHeartbeat() {
		m[GatewayHeartbeat] = "true"
	}

	if res.IsOneway() {
		m[GatewayOneway] = "true"
	}

	if res.MessageStatusType() == protocol.Error {
		m[GatewayMessageStatusType] = "Error"
	} else {
		m[GatewayMessageStatusType] = "Normal"
	}

	if res.CompressType() != protocol.Gzip {
		m["Content-Encoding"] = "gzip"
	}

	m[GatewayMeta] = urlencode(res.Metadata)
	m[GatewaySerializeType] = strconv.Itoa(int(res.SerializeType()))
	m[GatewayMessageID] = strconv.FormatUint(res.Seq(), 10)
	m[GatewayServicePath] = res.ServicePath
	m[GatewayServiceMethod] = res.ServiceMethod

	return m, res.Payload, nil
}

func urlencode(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}

	var buf bytes.Buffer
	for k, v := range data {
		buf.WriteString(url.QueryEscape(k))
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
		buf.WriteByte('&')
	}
	s := buf.String()
	return s[:len(s)-1]
}

// RegisterServerMessageChan registers the channel that receives server requests.
func (client *Client) RegisterServerMessageChan(ch chan<- *protocol.Message) {
	client.ServerMessageChan = ch
}

// UnregisterServerMessageChan removes ServerMessageChan.
func (client *Client) UnregisterServerMessageChan() {
	client.ServerMessageChan = nil
}

// IsClosing client is closing or not.
func (client *Client) IsClosing() bool {
	return client.closing
}

// IsShutdown client is shutdown or not.
func (client *Client) IsShutdown() bool {
	return client.shutdown
}

func (client *Client) Go(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	call := &Call{
		ServicePath:   servicePath,
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}

	meta := ctx.Value(share.ReqMetaDataKey)
	if meta != nil {
		call.Metadata = meta.(map[string]string)
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

	codec := share.Codecs[client.option.SerializeType]
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

	req := protocol.GetPoolMsg()
	req.SetMessageType(protocol.Request)
	req.SetSeq(seq)

	if call.ServicePath == "" && call.ServiceMethod == "" {
		req.SetHeartbeat(true)
	} else {
		req.SetSerializeType(client.option.SerializeType)
		if call.Metadata != nil {
			req.Metadata = call.Metadata
		}

		req.ServicePath = call.ServicePath
		req.ServiceMethod = call.ServiceMethod

		data, err := codec.Encode(call.Args)
		if err != nil {
			call.Error = err
			call.done()
			return
		}

		if len(data) > 1024 && client.option.CompressType == protocol.Gzip {
			data, err = util.Zip(data)
			if err != nil {
				call.Error = err
				call.done()
				return
			}

			req.SetCompressType(client.option.CompressType)
		}

		req.Payload = data
	}

	data := req.Encode()
	_, err := client.Conn.Write(data)
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

	protocol.FreeMsg(req)

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
	res := protocol.NewMessage()

	for err == nil {
		// res, err = protocol.Read(client.r)
		err = res.Decode(client.r)

		if err != nil {
			break
		}

		seq := res.Seq()
		var call *Call
		isServerMessage := (res.MessageType() == protocol.Request && !res.IsHeartbeat() && res.IsOneway())
		if !isServerMessage {
			client.mu.Lock()
			call = client.pending[seq]
			delete(client.pending, seq)
			client.mu.Unlock()
		}

		switch {
		case call == nil:
			if isServerMessage {
				if client.ServerMessageChan != nil {
					go client.handleServerRequest(res)
				}
				continue
			}
		case res.MessageStatusType() == protocol.Error:
			call.Error = ServiceError(res.Metadata[protocol.ServiceError])
			call.ResMetadata = res.Metadata

			if call.IsRaw {
				call.Metadata, call.Reply, _ = convertRes2Raw(res)
				call.Metadata[GatewayErrorMessage] = call.Error.Error()
			}
			call.done()
		default:
			if call.IsRaw {
				call.Metadata, call.Reply, _ = convertRes2Raw(res)
			} else {
				data := res.Payload
				if len(data) > 0 {
					if res.CompressType() == protocol.Gzip {
						data, err = util.Unzip(data)
						if err != nil {
							call.Error = ServiceError("unzip payload: " + err.Error())
						}
					}

					codec := share.Codecs[res.SerializeType()]
					if codec == nil {
						call.Error = ServiceError(ErrUnspportedCodec.Error())
					} else {
						err = codec.Decode(data, call.Reply)
						if err != nil {
							call.Error = ServiceError("decode payload: " + err.Error())
						}
					}
				}
				call.ResMetadata = res.Metadata
			}
			call.done()
		}

		res.Reset()
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
	if err != nil && err != io.EOF && !closing {
		log.Error("mrpc: client protocol error:", err)
	}
}

func (client *Client) handleServerRequest(msg *protocol.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("ServerMessageChan may be closed so client remove it. Please add it again if you want to handle server requests. error is %v", r)
			client.ServerMessageChan = nil
		}
	}()
	t := time.NewTimer(5 * time.Second)
	select {
	case client.ServerMessageChan <- msg:
	case <-t.C:
		log.Warnf("ServerMessageChan may be full so the server request %d has been dropped", msg.Seq())
	}
	t.Stop()
}

func (client *Client) heartbeat() {
	ticker := time.NewTicker(client.option.HeartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		if client.shutdown || client.closing {
			break
		}

		err := client.Call(context.Background(), "", "", nil, nil)
		if err != nil {
			log.Warnf("mrpc: client heartbeat error: %v to %s", err, client.Conn.RemoteAddr().String())
		}
	}
}
