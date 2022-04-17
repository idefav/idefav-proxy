package proxy

import (
	"bufio"
	"fmt"
	"idefav-proxy/cmd/mgr"
	"idefav-proxy/cmd/server"
	"idefav-proxy/cmd/upgrade"
	"idefav-proxy/pkg/socket"
	"log"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

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

	go func() {
		for {
			var conn = <-ConnC
			log.Println("Conn Close:", conn)
			conn.Close()
		}
	}()

	go inProxyServer.proc(ln)

	return nil
}

func (inProxyServer InProxyServer) Shutdown() error {
	for inProxyServer.NumOpen > 0 {
		time.Sleep(time.Second)
		continue
	}
	return nil
}

var ConnC chan net.Conn

func (inProxyServer InProxyServer) proc(ln net.Listener) error {
	for {
		conn, _ := ln.Accept()

		//log.Println("接收到新Http请求", err2)
		go func() {
			defer conn.Close()
			atomic.AddInt32(&inProxyServer.NumOpen, 1)
			defer atomic.AddInt32(&inProxyServer.NumOpen, -1)
			log.Printf("removeAddr: %s --> localAddr: %s", conn.RemoteAddr(), conn.LocalAddr())
			log.Println("conn:", &conn)

			var dst_host = "192.168.0.105:28080"
			dst, host, tcpConn, err := socket.GetOriginalDst(conn.(*net.TCPConn))
			log.Println(dst, host, tcpConn, err)
			if err == nil {
				dst_host = host
			}

			for {
				//log.Println("准备读取")
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				reader := bufio.NewReader(conn)
				peek, err := reader.Peek(4)
				if err != nil {
					//log.Println("连接断开")
					return
				}
				header := string(peek)

				if strings.HasPrefix(header, "GET") || strings.HasPrefix(header, "POST") {
					//log.Println("开始Http协议解析")
					inProxyServer.HttpProc(conn, reader, dst_host)
				} else {
					log.Println(header)
					writer := bufio.NewWriter(conn)
					var body = "收到!" + mgr.Version + "\n"
					var respContent = "HTTP/1.1 415 Unsupported Media Type\nServer: idefav\nContent-Type: text/html;charset=UTF-8\nContent-Length: " + strconv.Itoa(len(body)) + "\n\n" + body + "\n"
					_, err := writer.WriteString(respContent)
					if err != nil {
						log.Println(err)
					}
					//log.Println(count)
					writer.Flush()
					//c.Close()
				}
			}

		}()

	}
}

var Server = NewInProxyServer()

func init() {
	server.RegisterServer(Server)
}
