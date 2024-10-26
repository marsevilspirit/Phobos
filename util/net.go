package util

import (
	"net"
)

// GetFreePort 获取一个系统分配的空闲端口
func GetFreePort() (port int, err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	// 从监听器地址中提取出分配的端口
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
