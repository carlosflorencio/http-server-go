package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	Port = 4221
)

var (
	Directory string
)

func main() {
	if len(os.Args) == 3 {
		fmt.Println("Path received", os.Args[2])
		Directory = os.Args[2]
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", Port))
	if err != nil {
		log.Fatal("Failed to bind to port ", Port)
	}
	defer listener.Close()

	fmt.Println("Server listening on port ", Port)

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

	req, err := NewRequest(conn)
	if err != nil {
		fmt.Println("Error converting request: ", err.Error())
	}

	res := NewResponse(req)

	handlers(req, res)

	res.Send(conn)
}

func handlers(req *Request, res *Response) {
	echoPattern := regexp.MustCompile(`/echo/(.*)`)
	filesPattern := regexp.MustCompile(`/files/(.*)`)

	switch {
	case req.path == "/":
	case filesPattern.MatchString(req.path):
		params := filesPattern.FindStringSubmatch(req.path)

		if len(params) > 1 && len(params[1]) > 0 {

			if req.method == "GET" {
				res.ServeFile(params[1])
			} else {
				res.WriteFile(params[1], req.body)
			}
		} else {
			res.status = 404
		}
	case req.path == "/user-agent":
		res.body = []byte(req.headers.Get("User-Agent"))
	case echoPattern.MatchString(req.path):
		params := echoPattern.FindStringSubmatch(req.path)

		if len(params) > 1 && len(params[1]) > 0 {
			res.body = []byte(params[1])
		}
	default:
		res.status = 404
	}
}

type Request struct {
	proto   string
	method  string
	path    string
	body    []byte
	headers textproto.MIMEHeader
}

func NewRequest(conn net.Conn) (*Request, error) {
	reader := bufio.NewReader(conn)
	text := textproto.NewReader(reader)

	req := new(Request)

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

	if len(req.headers["Content-Length"]) == 0 {
		return nil, errors.New("no content length")
	}

	length, err := strconv.Atoi(req.headers["Content-Length"][0])
	if err != nil {
		return nil, err
	}

	body := make([]byte, length)
	if _, err = io.ReadFull(reader, body); err != nil {
		return nil, err
	}

	req.body = body

	return req, nil
}

type Response struct {
	req     *Request
	status  int
	body    []byte
	headers textproto.MIMEHeader
}

func NewResponse(req *Request) *Response {
	return &Response{
		req: req,
		headers: textproto.MIMEHeader{
			"Content-Type": []string{"text/plain"},
		},
		status: 200,
	}
}

func (r *Response) Send(conn net.Conn) {
	payload := r.req.proto + " " + strconv.Itoa(r.status) + " " + r.StatusString() + "\r\n"

	for key, value := range r.headers {
		payload += key + ": " + value[0] + "\r\n"
	}

	payload += "Content-Length: " + strconv.Itoa(len(r.body)) + "\r\n\r\n" + string(r.body)

	conn.Write([]byte(payload))
}

func (r *Response) StatusString() string {
	switch r.status {
	case 200:
		return "OK"
	case 404:
		return "Not Found"
	default:
		return "Unknown"
	}
}

func (r *Response) ServeFile(fileName string) {
	path := filepath.Join(Directory, fileName)

	_, err := os.Stat(path)
	if err != nil {
		r.status = 404
		return
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		r.status = 404
		return
	}

	r.headers["Content-Type"] = []string{"application/octet-stream"}
	r.body = contents
}

func (r *Response) WriteFile(fileName string, contents []byte) {
	path := filepath.Join(Directory, fileName)

	err := os.WriteFile(path, contents, 0666)
	if err != nil {
		r.status = 500
		r.body = []byte(err.Error())
		return
	}

	r.status = 201
}
