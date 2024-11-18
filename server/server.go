package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/marsevilspirit/m_RPC/log"
	"github.com/marsevilspirit/m_RPC/protocol"
	"github.com/marsevilspirit/m_RPC/share"
)

var ErrServerClosed = errors.New("http: Server closed")

const (
	ReaderBufferSize = 1024
	WriterBufferSize = 1024
)

type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "mrpc context value " + k.name
}

var (
	RemoteConnContextKey = &contextKey{"remote-conn"}
)

type Server struct {
	ln           net.Listener
	readTimeout  time.Duration
	writeTimeout time.Duration

	serviceMapMu sync.RWMutex
	serviceMap   map[string]*service

	mu         sync.Mutex
	activeConn map[net.Conn]struct{}
	done       chan struct{}

	// inShutdown int32
	onShutdown []func()

	tlsConfig *tls.Config

	options map[string]interface{}

	Plugins PluginContainer

	AuthFunc func(ctx context.Context, req *protocol.Message, token string) error
}

func NewServer(options ...OptionFn) *Server {
	s := &Server{
		Plugins: &pluginContainer{},
		options: make(map[string]interface{}),
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

func (s *Server) Address() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ln == nil {
		return nil
	}

	return s.ln.Addr()
}

func (s *Server) getDone() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done == nil {
		s.done = make(chan struct{})
	}

	return s.done
}

func (s *Server) Serve(network, address string) (err error) {
	var ln net.Listener

	ln, err = s.makeListener(network, address)
	if err != nil {
		return err
	}
	log.Info("serving on ", ln.Addr().String())

	if network == "http" {
		s.serveByHTTP(ln, "")
		return nil
	}

	return s.serveListener(ln)
}

func (s *Server) serveListener(ln net.Listener) error {
	if s.Plugins == nil {
		s.Plugins = &pluginContainer{}
	}

	var tempDelay time.Duration

	s.mu.Lock()
	s.ln = ln
	if s.activeConn == nil {
		s.activeConn = make(map[net.Conn]struct{})
	}
	s.mu.Unlock()

	for {
		conn, e := ln.Accept()
		if e != nil {
			select {
			case <-s.getDone():
				return ErrServerClosed
			default:
			}

			if ne, ok := e.(interface{ Temporary() bool }); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				log.Errorf("mrpc: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0

		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(3 * time.Minute)
		}

		s.mu.Lock()
		s.activeConn[conn] = struct{}{}
		s.mu.Unlock()

		conn, ok := s.Plugins.DoPostConnAccept(conn)
		if !ok {
			continue
		}

		go s.serveConn(conn)
	}
}

func (s *Server) serveByHTTP(ln net.Listener, rpcPath string) {
	s.ln = ln

	if s.Plugins == nil {
		s.Plugins = &pluginContainer{}
	}

	if rpcPath == "" {
		rpcPath = share.DefaultRPCPath
	}
	http.Handle(rpcPath, s)
	srv := &http.Server{Handler: nil}

	s.mu.Lock()
	if s.activeConn == nil {
		s.activeConn = make(map[net.Conn]struct{})
	}
	s.mu.Unlock()

	srv.Serve(ln)
}

func (s *Server) serveConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)

			ss := runtime.Stack(buf, false)

			if ss > size {
				ss = size
			}

			buf = buf[:ss]

			log.Errorf("serving %s panic error: %s, stack:\n %s", conn.RemoteAddr(), err, buf)
		}

		s.mu.Lock()
		delete(s.activeConn, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	if tlsConn, ok := conn.(*tls.Conn); ok {
		if d := s.readTimeout; d != 0 {
			conn.SetReadDeadline(time.Now().Add(d))
		}
		if d := s.writeTimeout; d != 0 {
			conn.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			log.Errorf("mrpc: TLS handshake error from %s: %v", conn.RemoteAddr(), err)
		}
	}

	ctx := context.WithValue(context.Background(), RemoteConnContextKey, conn)
	r := bufio.NewReaderSize(conn, ReaderBufferSize)

	for {
		now := time.Now()

		if s.readTimeout != 0 {
			conn.SetReadDeadline(now.Add(s.readTimeout))
		}

		req, err := s.readRequest(ctx, r)
		if err != nil {
			if err == io.EOF {
				log.Infof("client disconnected: %s", conn.RemoteAddr().String())
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				log.Infof("rpcx: connection %s is closed", conn.RemoteAddr().String())
			} else {
				log.Warnf("mrpc: failed to read request: %v", err)
			}
			return
		}

		if s.writeTimeout != 0 {
			conn.SetWriteDeadline(now.Add(s.writeTimeout))
		}

		err = s.auth(ctx, req)
		if err != nil {
			s.Plugins.DoPreWriteResponse(ctx, req)
			if !req.IsOneway() {
				res := req.Clone()
				res.SetMessageType(protocol.Response)
				handleError(res, err)
				data := res.Encode()
				conn.Write(data)
				s.Plugins.DoPostWriteResponse(ctx, req, res, err)
				protocol.FreeMsg(res)
			}
			protocol.FreeMsg(req)
			continue
		}

		go func() {
			if req.IsHeartbeat() {
				req.SetMessageType(protocol.Response)
				data := req.Encode()
				conn.Write(data)
				return
			}

			resMetadata := make(map[string]string)
			newCtx := context.WithValue(context.WithValue(ctx, share.ReqMetaDataKey, req.Metadata), share.ResMetaDataKey, resMetadata)

			res, err := s.handleRequest(newCtx, req)
			if err != nil {
				log.Warnf("mrpc: failed to handle request: %v", err)
			}

			s.Plugins.DoPreWriteResponse(newCtx, req)

			if !req.IsOneway() {
				if len(resMetadata) > 0 {
					meta := res.Metadata
					if meta == nil {
						res.Metadata = resMetadata
					} else {
						for k, v := range resMetadata {
							meta[k] = v
						}
					}
				}

				data := res.Encode()
				conn.Write(data)
			}

			s.Plugins.DoPostWriteResponse(newCtx, req, res, err)

			protocol.FreeMsg(req)
			protocol.FreeMsg(res)
		}()
	}
}

func (s *Server) readRequest(ctx context.Context, r io.Reader) (req *protocol.Message, err error) {
	s.Plugins.DoPreReadRequest(ctx)

	req = protocol.GetPoolMsg()
	err = req.Decode(r)

	s.Plugins.DoPostReadRequest(ctx, req, err)

	return req, err
}

func (s *Server) auth(ctx context.Context, req *protocol.Message) (err error) {
	// 验证身份
	if s.AuthFunc != nil {
		token := req.Metadata[share.AuthKey]
		return s.AuthFunc(ctx, req, token)
	}

	return nil
}

func (s *Server) handleRequest(ctx context.Context, req *protocol.Message) (res *protocol.Message, err error) {
	serviceName := req.ServicePath
	methodName := req.ServiceMethod

	res = req.Clone()
	res.SetMessageType(protocol.Response)

	s.serviceMapMu.RLock()
	service := s.serviceMap[serviceName]
	s.serviceMapMu.RUnlock()
	if service == nil {
		err = errors.New("mrpc: can't find service " + serviceName)
		return handleError(res, err)
	}
	mtype := service.method[methodName]
	if mtype == nil {
		if service.function[methodName] != nil {
			return s.handleRequestForFunction(ctx, req)
		}
		err = errors.New("mrpc: can't find method " + methodName)
		return handleError(res, err)
	}

	var argv, replyv reflect.Value

	argIsValue := false
	if mtype.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}

	codec := share.Codecs[req.SerializeType()]
	if codec == nil {
		err = fmt.Errorf("can not find codec for %d", req.SerializeType())
		return handleError(res, err)
	}

	err = codec.Decode(req.Payload, argv.Interface())
	if err != nil {
		return handleError(res, err)
	}

	if argIsValue {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.ReplyType.Elem())

	err = service.call(ctx, mtype, argv, replyv)
	if err != nil {
		return handleError(res, err)
	}

	if !req.IsOneway() {
		data, err := codec.Encode(replyv.Interface())
		if err != nil {
			return handleError(res, err)

		}
		res.Payload = data
	}

	return res, nil
}

func (s *Server) handleRequestForFunction(ctx context.Context, req *protocol.Message) (res *protocol.Message, err error) {
	res = req.Clone()

	res.SetMessageType(protocol.Response)

	serviceName := req.ServicePath
	methodName := req.ServiceMethod

	s.serviceMapMu.RLock()
	service := s.serviceMap[serviceName]
	s.serviceMapMu.RUnlock()

	if service == nil {
		err = errors.New("mrpc: can't find the default service")
		return handleError(res, err)
	}

	mtype := service.function[methodName]
	if mtype == nil {
		err = errors.New("mrpc: can't find method " + methodName)
		return handleError(res, err)
	}

	var argv, replyv reflect.Value

	argIsValue := false
	if mtype.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}

	codec := share.Codecs[req.SerializeType()]
	if codec == nil {
		err = fmt.Errorf("can not find codec for %d", req.SerializeType())
		return handleError(res, err)
	}

	err = codec.Decode(req.Payload, argv.Interface())
	if err != nil {
		return handleError(res, err)
	}

	if argIsValue {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.ReplyType.Elem())

	err = service.callForFunction(ctx, mtype, argv, replyv)
	if err != nil {
		return handleError(res, err)
	}

	if !req.IsOneway() {
		data, err := codec.Encode(replyv.Interface())
		if err != nil {
			return handleError(res, err)
		}
		res.Payload = data
	}

	return res, nil
}

func handleError(res *protocol.Message, err error) (*protocol.Message, error) {
	res.SetMessageStatusType(protocol.Error)
	if res.Metadata == nil {
		res.Metadata = make(map[string]string)
	}
	res.Metadata[protocol.ServiceError] = err.Error()
	return res, err
}

var connected = "200 Connected to mrpc"

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 must CONNECT\n")
		return
	}
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Info("rpc hijacking", req.RemoteAddr, ": ", err.Error())
		return
	}
	io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")

	s.mu.Lock()
	s.activeConn[conn] = struct{}{}
	s.mu.Unlock()

	s.serveConn(conn)
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closeDoneLocked()

	var err error
	if s.ln != nil {
		err = s.ln.Close()
	}

	for c := range s.activeConn {
		c.Close()
		delete(s.activeConn, c)
	}

	return err
}

// 优雅关闭连接
func (s *Server) RegisterOnShutdown(f func()) {
	s.mu.Lock()
	s.onShutdown = append(s.onShutdown, f)
	s.mu.Unlock()
}

func (s *Server) closeDoneLocked() {
	ch := s.getDoneLocked()
	select {
	case <-ch:
		// 已经关闭，不用再次关闭
	default:
		close(ch)
	}
}

func (s *Server) getDoneLocked() chan struct{} {
	if s.done == nil {
		s.done = make(chan struct{})
	}

	return s.done
}

func (s *Server) HandleHTTP(rpcPath, debugPath string) {
	http.Handle(rpcPath, s)
}
