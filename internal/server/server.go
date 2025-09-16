package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync/atomic"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
)

type Server struct {
	port     uint16
	listener net.Listener
	closed   atomic.Bool
	handler Handler
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

	req, err := request.RequestFromReader(conn)
	fmt.Println(req, err)
	if err != nil {
		return
	}

	bodyReader := req.Body()
	defer bodyReader.Close()

	b, e := io.ReadAll(bodyReader)
	fmt.Println("Body:", string(b), "Error:", e)

	buffer := new(bytes.Buffer)
	handlerError := s.handler(buffer, req)
	if handlerError != nil {
		conn.Write(handlerError.response().Bytes())
		return
	} 

	resp := response.NewResponse().WithBody(buffer.Bytes())
	conn.Write(resp.Bytes())
}

func newServer(port uint16, handler Handler) (*Server, error) {
	return &Server{
		port: port,
		handler: handler,
	}, nil
}

func Serve(port uint16, handler Handler) (*Server, error) {
	s, err := newServer(port, handler)
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
