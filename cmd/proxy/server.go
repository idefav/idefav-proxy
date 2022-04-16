package proxy

import (
	"idefav-proxy/pkg/pool"
	"net"
	"time"
)

type InProxyServer struct {
	Connections map[string]net.Conn
	NumOpen     int32
	IdleTimeOut time.Duration
	ConnPool    pool.Pool
}

func NewInProxyServer() *InProxyServer {
	connPool, _ := NewConnPool("192.168.0.105", 28080, 1, 10000, 10000)
	return &InProxyServer{
		Connections: make(map[string]net.Conn),
		NumOpen:     0,
		IdleTimeOut: 60 * 1000,
		ConnPool:    connPool,
	}
}
