package server

import (
	"crypto/tls"
	"fmt"
	"net"

	reuseport "github.com/kavu/go_reuseport"
)

var makeListeners = make(map[string]MakeListener)

func init() {
	makeListeners["tcp"] = tcpMakeListener
	makeListeners["http"] = tcpMakeListener
	makeListeners["reuseport"] = reuseportMakeListener
}

func RegisterListener(network string, ml MakeListener) {
	makeListeners[network] = ml
}

type MakeListener func(s *Server, address string) (ln net.Listener, err error)

// validIP4 函数用于检测一个地址是否为有效的 IPv4 地址
func validIP4(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() != nil
}

func (s *Server) makeListener(network, address string) (ln net.Listener, err error) {
	ml := makeListeners[network]
	if ml == nil {
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	return ml(s, address)
}

func tcpMakeListener(s *Server, address string) (ln net.Listener, err error) {
	if s.tlsConfig == nil {
		ln, err = net.Listen("tcp", address)
	} else {
		ln, err = tls.Listen("tcp", address, s.tlsConfig)
	}

	return ln, err
}

func reuseportMakeListener(s *Server, address string) (ln net.Listener, err error) {
	var network string

	if validIP4(address) {
		network = "tcp4"
	} else {
		network = "tcp6"
	}

	ln, err = reuseport.Listen(network, address)

	return ln, err
}
