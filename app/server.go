package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", "localhost:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Server listening on port 4221")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	req, err := convertRequest(conn)
	if err != nil {
		fmt.Println("Error converting request: ", err.Error())
	}

	if req.path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

type request struct {
	proto   string
	method  string
	path    string
	body    []byte
	headers textproto.MIMEHeader
}

func convertRequest(conn net.Conn) (*request, error) {
	reader := bufio.NewReader(conn)
	text := textproto.NewReader(reader)

	req := new(request)

	// parse first line, GET / HTTP/1.1
	line, err := text.ReadLine()
	if err != nil {
		fmt.Println("Error parsing first line: ", err.Error())
		return nil, err
	}

	parts := strings.Split(line, " ")
	req.method = parts[0]
	req.path = parts[1]
	req.proto = parts[2]

	// parse headers
	req.headers, err = text.ReadMIMEHeader()
	if err != nil {
		fmt.Println("Error parsing headers: ", err.Error())
		return nil, err
	}

	if req.method == "GET" || req.method == "HEAD" {
		return req, nil
	}

	return nil, errors.New("Unsupported method: " + req.method)
}
