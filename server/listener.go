package server

import (
	"net"

	reuseport "github.com/kavu/go_reuseport"
)

// validIP4 函数用于检测一个地址是否为有效的 IPv4 地址
func validIP4(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() != nil
}

func makeListener(network, address string) (ln net.Listener, err error) {
	switch network {
	case "reuseport":
		if validIP4(address) {
			network = "tcp4"
		} else {
			network = "tcp6"
		}

		ln, err = reuseport.NewReusablePortListener(network, address)

	default:
		ln, err = net.Listen(network, address)
	}

	return ln, err
}
