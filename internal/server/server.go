package server

import (
	"fmt"
	"log"
	"net"
	"runtime/debug"
	"sync/atomic"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
)

type Server struct {
	port     uint16
	listener net.Listener
	closed   atomic.Bool
	handler  Handler
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
	defer func() {
		if r := recover(); r != nil {
			log.Println("recovered from panic:", r)
			debug.PrintStack()
			resp := response.NewTextResponse(response.GetStatusReason(response.StatusInternalServerError)).WithStatusCode(response.StatusInternalServerError)
			resp.Write(conn)
			conn.Close()
			return
		}

		// todo remove when keep alive is used
		if conn != nil {
			conn.Close()
		}
	}()

	req, err := request.RequestFromReader(conn)
	// fmt.Println(req, err)
	if err != nil {
		response.NewBaseResponse().WithStatusCode(400).Write(conn)
		return
	}

	// bodyReader := req.Body()
	// defer bodyReader.Close()

	// b, e := io.ReadAll(bodyReader)
	// fmt.Println("Body:", string(b), "Error:", e)

	resp := s.handler(req)
	err = resp.Write(conn)
	if err != nil {
		fmt.Println("resp writing error", err)
	}
}

func newServer(port uint16, handler Handler) (*Server, error) {
	return &Server{
		port:    port,
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
