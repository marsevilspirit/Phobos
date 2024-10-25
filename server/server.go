package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/marsevilspirit/m_RPC/log"
	"github.com/marsevilspirit/m_RPC/protocol"
)

var ErrServerClosed = errors.New("http: Server closed")

const (
	DefaultRPCPath = "/__mrpc__"

	ReaderBufferSize = 16 * 1024
	WriterBufferSize = 16 * 1024

	ServicePath   = "__mrpc_path__"
	ServiceMethod = "__mrpc_method__"
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
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	serviceMapMu sync.RWMutex
	serviceMap   map[string]*service
	methodMap    map[string]*methodType

	mu         sync.Mutex
	activeConn map[net.Conn]struct{}
	done       chan struct{}
}

func (s *Server) getDone() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done == nil {
		s.done = make(chan struct{})
	}

	return s.done
}

func (s *Server) idleTimeout() time.Duration {
	if s.IdleTimeout != 0 {
		return s.IdleTimeout
	}

	return s.ReadTimeout
}

func (s *Server) Serve(ln net.Listener) error {
	log.Info("serving")

	s.ln = ln

	var tempDelay time.Duration

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.getDone():
				return ErrServerClosed
			default:
			}

			// 检查是不是暂时错误
			if ne, ok := err.(interface{ Temporary() bool }); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				log.Errorf("mrpc: Accept error: %v; retrying in %v", err, tempDelay)

				continue
			}

			log.Errorf("mrpc: done serving; accepct = %v", err)

			return err
		}

		tempDelay = 0

		if tcp, ok := conn.(*net.TCPConn); ok {
			tcp.SetKeepAlive(true)
			tcp.SetKeepAlivePeriod(3 * time.Minute)
		}

		s.mu.Lock()
		s.activeConn[conn] = struct{}{}
		s.mu.Unlock()

		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Errorf("serving %s panic error: %s, stack:\n %s", conn.RemoteAddr(), err, buf)
		}

		s.mu.Lock()
		delete(s.activeConn, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	if tlsConn, ok := conn.(*tls.Conn); ok {
		if d := s.ReadTimeout; d != 0 {
			conn.SetReadDeadline(time.Now().Add(d))
		}
		if d := s.WriteTimeout; d != 0 {
			conn.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			log.Errorf("mrpc: TLS handshake error from %s: %v", conn.RemoteAddr(), err)
		}
	}

	ctx := context.WithValue(context.Background(), RemoteConnContextKey, conn)
	r := bufio.NewReaderSize(conn, ReaderBufferSize)
	w := bufio.NewWriterSize(conn, WriterBufferSize)

	for {
		now := time.Now()

		if s.IdleTimeout != 0 {
			conn.SetReadDeadline(now.Add(s.IdleTimeout))
		}

		if s.ReadTimeout != 0 {
			conn.SetReadDeadline(now.Add(s.ReadTimeout))
		}

		if s.WriteTimeout != 0 {
			conn.SetWriteDeadline(now.Add(s.WriteTimeout))
		}

		req, err := s.readRequest(ctx, r)
		if err != nil {
			log.Errorf("mrpc: failed to read request: %v", err)
		}

		go func() {
			res, err := s.handleRequest(ctx, req)
			if err != nil {
				log.Errorf("mrpc: failed to handle request: %v", err)
			}

			res.WriteTo(w)
		}()
	}
}

func (s *Server) readRequest(ctx context.Context, r io.Reader) (req *protocol.Message, err error) {
	req, err = protocol.Read(r)
	return req, err
}

func (s *Server) handleRequest(ctx context.Context, req *protocol.Message) (res *protocol.Message, err error) {
	res = protocol.NewMessage()

	serviceName := req.Metadata[ServicePath]
	methodName := req.Metadata[ServiceMethod]

	s.serviceMapMu.RLock()
	service := s.serviceMap[serviceName]
	s.serviceMapMu.RUnlock()
	if service == nil {
		err = errors.New("mrpc: can't find service " + serviceName)
		return
	}
	mtype := service.method[methodName]
	if mtype == nil {
		err = errors.New("mrpc: can't find method " + methodName)
	}

	var argv, replyv reflect.Value

	argIsValue := false
	if mtype.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}

	// TODO: decode from payload

	if argIsValue {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.ReplyType.Elem())

	err = service.call(ctx, mtype, argv, replyv)
	if err != nil {
		// TODO: set error response
		return res, err
	}

	// TODO: clone req for req,
	// encode replyv to res.Payload or
	// return res

	return res, nil
}

var connected = "200 Connected to Go RPC"

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
	s.serveConn(conn)
}

func (s *Server) HandleHTTP(rpcPath, debugPath string) {
	http.Handle(rpcPath, s)
}
