package http

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
)

func Resolve(conn net.Conn, header string, reader *bufio.Reader) error {
	var headers = make([]string, 0)
	headers = append(headers, header)
	scanner := bufio.NewScanner(reader)
	for {
		scan := scanner.Scan()
		if scan {
			headers = append(headers, scanner.Text())
			if scanner.Text() == "" {
				break
			}
		} else {
			break
		}

	}

	log.Println(headers)

	writer := bufio.NewWriter(conn)
	destConn, err2 := net.Dial("tcp", "192.168.0.105:28080")
	if err2 != nil {
		var body = ""
		body = err2.Error()
		var respContent = "HTTP/1.1 200 OK\nServer: idefav\nContent-Type: text/plain;charset=UTF-8\nContent-Length: " + strconv.Itoa(len(body)) + "\n\n" + body + "\n"
		writer.Write([]byte(respContent))
		return err2
	}

	var headerStr = strings.Join(headers, "\n")
	_, err2 = destConn.Write([]byte(headerStr + "\r\n"))

	var bytes = make([]byte, 1024)
	n, _ := reader.Read(bytes)
	destConn.Write(bytes[:n])
	go func() {

		reader.WriteTo(destConn)
	}()
	respReader := bufio.NewReader(destConn)
	line, err2 := respReader.ReadSlice('\n')
	conn.Write(line)
	conn.Write([]byte("Server: idefav-proxy\n"))
	respReader.WriteTo(conn)
	//io.Copy(conn, destConn)

	return err2

}
