package server

import (
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
)

type Server struct {
	opts     ServerOpts
	listener net.Listener
	closed   atomic.Bool
	handler  Handler
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

func (s *Server) listen() error {
	listener, err := net.Listen("tcp", s.opts.Address)
	if err != nil {
		return err
	}
	s.listener = listener

	for {
		conn, err := s.listener.Accept()
		if err != nil && !s.closed.Load() {
			panic("unable to accept connection: " + err.Error())
		}

		if s.opts.ReadTimeout != 0 {
			if conn != nil {
				conn.SetReadDeadline(time.Now().Add(s.opts.ReadTimeout))
			}
		}
		if s.opts.WriteTimeout != 0 {
			if conn != nil {
				conn.SetWriteDeadline(time.Now().Add(s.opts.WriteTimeout))
			}
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			resp := s.opts.Recovery(r)
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
	hostHeader := req.Headers.Get("host")
	if hostHeader == "" || len(strings.Split(hostHeader, ",")) > 1 {
		response.NewBaseResponse().WithStatusCode(400).Write(conn)
		return
	}

	// bodyReader := req.Body()
	// defer bodyReader.Close()

	// b, e := io.ReadAll(bodyReader)
	// fmt.Println("Body:", string(b), "Error:", e)

	resp := s.handler(req)
	if dateHeader := resp.GetHeaders().Get(""); dateHeader == "" {
		resp.WithHeader("date", time.Now().Format(time.RFC1123))
	}
	err = resp.Write(conn)
	if err != nil {
		log.Println("unable to write response to connection:", err)
	}
}

func newServer(opts ServerOpts, handler Handler) (*Server, error) {
	if opts.Recovery == nil {
		opts.Recovery = defaultRecovery
	}
	return &Server{
		opts:    opts,
		handler: handler,
	}, nil
}

func Serve(opts ServerOpts, handler Handler) (*Server, error) {
	s, err := newServer(opts, handler)
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
