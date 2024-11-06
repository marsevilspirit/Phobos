package server

import (
	"crypto/tls"
	"net"

	reuseport "github.com/kavu/go_reuseport"
)

// validIP4 函数用于检测一个地址是否为有效的 IPv4 地址
func validIP4(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() != nil
}

func (s *Server) makeListener(network, address string) (ln net.Listener, err error) {
	switch network {
	case "reuseport":
		if validIP4(address) {
			network = "tcp4"
		} else {
			network = "tcp6"
		}

		ln, err = reuseport.NewReusablePortListener(network, address)

	default: // tcp, http
		if s.TLSConfig != nil {
			ln, err = net.Listen(network, address)
			if err != nil {
				return nil, err
			}

			ln = tls.NewListener(ln, s.TLSConfig)
		} else {
			ln, err = net.Listen(network, address)
		}
	}

	return ln, err
}
