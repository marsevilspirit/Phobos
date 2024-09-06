package mrpc

import (
	"context"
	"errors"
	"log"
	"net"
	"reflect"
	"sync"

	//"github.com/marsevilspirit/m_RPC/codec"
)

type methodHandler func(srv any, ctx context.Context, dec func(any) error) (any, error)

// MethodDesc represents an RPC service's method specification.
type MethodDesc struct {
	MethodName string
	Handler    methodHandler
}

// ServiceDesc represents an RPC service's specification.
type ServiceDesc struct {
	ServiceName string
	// The pointer to the service interface. Used to check whether the user
	// provided implementation satisfies the interface requirements.
	HandlerType any
	Methods     []MethodDesc
	Metadata    any
}

type serviceInfo struct {
	serviceImpl any
	methods     map[string]*MethodDesc
	mdata       any
}

type Server struct {
	opts serverOptions

	mu sync.Mutex

	lis      map[net.Listener]bool // net.Listener
	serve    bool                  // true if the server is serving
	cv       *sync.Cond
	services map[string]*serviceInfo

	serveWG   sync.WaitGroup
	handlesWG sync.WaitGroup
}

type serverOptions struct {
	//codec codec.ProtobufCodec
}

func NewServer(opt ...serverOptions) *Server {
	s := &Server{
		lis:      make(map[net.Listener]bool),
		serve:    false,
		services: make(map[string]*serviceInfo),
	}

	s.cv = sync.NewCond(&s.mu)

	return s
}

// ServiceRegistrar wraps a single method that supports service registration. It
// enables users to pass concrete types other than mrpc.Server to the service
// registration methods exported by the IDL generated code.
type ServiceRegistrar interface {
	// RegisterService registers a service and its implementation to the
	// concrete type implementing this interface.  It may not be called
	// once the server has started serving.
	// desc describes the service and its methods and handlers. impl is the
	// service implementation which is passed to the method handlers.
	RegisterService(desc *ServiceDesc, impl any)
}

func (s *Server) RegisterService(sd *ServiceDesc, ss any) {
	if ss != nil {
		ht := reflect.TypeOf(sd.HandlerType).Elem()
		st := reflect.TypeOf(ss)
		if !st.Implements(ht) {
			log.Fatalf("mrpc: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht)
		}
	}
	s.register(sd, ss)
}

func (s *Server) register(sd *ServiceDesc, ss any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.serve {
		log.Fatalf("mrpc: Server.RegisterService after Server.Serve for %q", sd.ServiceName)
	}
	if _, ok := s.services[sd.ServiceName]; ok {
		log.Fatalf("mrpc: Server.RegisterService found duplicate service registration for %q", sd.ServiceName)
	}

	si := &serviceInfo{
		serviceImpl: ss,
		methods:     make(map[string]*MethodDesc),
		mdata:       sd.Metadata,
	}

	for _, md := range sd.Methods {
		si.methods[md.MethodName] = &md
	}

	s.services[sd.ServiceName] = si
}

func (s *Server) Serve(lis net.Listener) error {
	s.mu.Lock()
	s.serve = true
	if s.lis == nil {
		s.mu.Unlock()
		lis.Close()
		return errors.New("mrpc: Server.Serve called with nil listener")
	}

	s.serveWG.Add(1)
	defer s.serveWG.Done()

	s.lis[lis] = true

	defer func() {
		s.mu.Lock()
		if s.lis != nil && s.lis[lis] {
			lis.Close()
			delete(s.lis, lis)
		}
		s.mu.Unlock()
	}()

	s.mu.Unlock()

	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}

		s.serveWG.Add(1)
		go func() {
			defer s.serveWG.Done()
			s.handleConn(lis.Addr().String(), conn)
		}()
	}
}

func (s *Server) handleConn(addr string, conn net.Conn) {	
	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println("conn.Read error:", err)
			return
		} 

		log.Println("conn.Read buffer:", buffer[:n])

		data := buffer[:n]

		log.Printf("data from %v: %v\n", addr, data)
	}
}
