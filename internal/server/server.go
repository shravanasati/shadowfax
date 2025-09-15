package server

import (
	"fmt"
	"io"
	"net"
	"sync/atomic"

	"github.com/shravanasati/shadowfax/internal/request"
)

type Server struct {
	port     uint16
	listener net.Listener
	closed   atomic.Bool
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

func (s *Server) listen() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	s.listener = listener

	for {
		conn, err := s.listener.Accept()
		if err != nil && !s.closed.Load() {
			panic("unable to accept connection: " + err.Error())
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	resp := []byte(`HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 12

Hello World!`)

	fmt.Println("waiting for request parsing")
	req, err := request.RequestFromReader(conn)
	fmt.Println(req, err)
	if err != nil {
		return
	}

	bodyReader := req.Body()
	defer bodyReader.Close()

	b, e := io.ReadAll(bodyReader)
	fmt.Println("Body:", string(b), "Error:", e)
	conn.Write(resp)
}

func newServer(port uint16) (*Server, error) {
	return &Server{
		port: port,
	}, nil
}

func Serve(port uint16) (*Server, error) {
	s, err := newServer(port)
	if err != nil {
		return nil, err
	}

	go func() {
		err := s.listen()
		if err != nil {
			panic(err)
		}
	}()
	return s, nil
}
