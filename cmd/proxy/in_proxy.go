package proxy

import (
	"bufio"
	"fmt"
	"idefav-proxy/cmd/mgr"
	idefavHttp "idefav-proxy/cmd/proxy/http"
	"idefav-proxy/cmd/server"
	"idefav-proxy/cmd/upgrade"
	"log"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type InProxyServer struct {
	Connections map[string]net.Conn
	numOpen     int32
}

func NewInProxyServer() *InProxyServer {
	return &InProxyServer{Connections: make(map[string]net.Conn), numOpen: 0}
}
func generateKey(localAddr, remoteAddr string) string {
	return fmt.Sprintf("%s-%s", localAddr, remoteAddr)
}

func (inProxyServer InProxyServer) AddConn(conn net.Conn) error {
	return nil
}

func (inProxyServer InProxyServer) RemoveConn(conn net.Conn) error {
	return nil
}

func (inProxyServer InProxyServer) Startup() error {
	ln, err := upgrade.Upgrade.Listen("tcp", ":15006")
	if err != nil {
		return err
	}

	go inProxyServer.proc(ln)
	return nil
}

func (inProxyServer InProxyServer) Shutdown() error {
	for inProxyServer.numOpen > 0 {
		time.Sleep(time.Second)
		continue
	}
	return nil
}

func (inProxyServer *InProxyServer) proc(ln net.Listener) error {
	for {
		conn, _ := ln.Accept()
		go func() {
			defer conn.Close()
			atomic.AddInt32(&inProxyServer.numOpen, 1)
			defer atomic.AddInt32(&inProxyServer.numOpen, -1)

			reader := bufio.NewReader(conn)
			proto, _, err := reader.ReadLine()
			if err != nil {
				log.Println(err)
				return
			}
			//request, err := http.ReadRequest(reader)

			var header = string(proto)

			if strings.Contains(header, "HTTP") {
				idefavHttp.Resolve(conn, header, reader)
			} else {
				var body = "收到!" + mgr.Version + "\n"
				var respContent = "HTTP/1.1 200 OK\nServer: idefav\nContent-Type: text/html;charset=UTF-8\nContent-Length: " + strconv.Itoa(len(body)) + "\n\n" + body + "\n"
				writer := bufio.NewWriterSize(conn, len(respContent))
				count, err := writer.WriteString(respContent)
				if err != nil {
					log.Println(err)
				}
				log.Println(count)
				writer.Flush()
			}

		}()

	}
}

func init() {
	proxyServer := NewInProxyServer()
	server.RegisterServer(proxyServer)
}
