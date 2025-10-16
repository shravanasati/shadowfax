package server

import (
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
)

type Server struct {
	opts     ServerOpts
	listener net.Listener
	closed   atomic.Bool
	handler  Handler
}

// Shutdown the server.
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
		if err != nil {
			if !s.closed.Load() {
				log.Println("unable to accept connection: " + err.Error())
				return err
			}
			// If server is closed, break out of the loop
			break
		}

		// Ensure connection is not nil before proceeding
		if conn == nil {
			continue
		}

		if s.opts.ReadTimeout != 0 {
			conn.SetReadDeadline(time.Now().Add(s.opts.ReadTimeout))
		}
		if s.opts.WriteTimeout != 0 {
			conn.SetWriteDeadline(time.Now().Add(s.opts.WriteTimeout))
		}
		go s.handle(conn)
	}
	return nil
}

func (s *Server) handle(conn net.Conn) {
	shouldCloseConn := false
	if s.opts.KeepAliveTimeout == 0 {
		shouldCloseConn = true
	}

	// defers are stacked

	defer func() {
		if conn != nil && shouldCloseConn {
			if err := conn.Close(); err != nil {
				log.Println("unable to close connection", err)
			}
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			resp := s.opts.Recovery(r)
			resp.Write(conn)
			conn.Close()
			return
		}
	}()

	for {
		if s.opts.KeepAliveTimeout != 0 {
			if conn != nil {
				conn.SetDeadline(time.Now().Add(s.opts.KeepAliveTimeout))
			}
		}

		badReqResponse := response.NewBaseResponse().WithStatusCode(response.StatusBadRequest)
		req, err := request.RequestFromReader(conn)
		if err != nil {
			// invalid request
			badReqResponse.Write(conn)
			shouldCloseConn = true
			break
		}

		hostHeader := req.Headers.Get("host")
		if hostHeader == "" || len(strings.Split(hostHeader, ",")) > 1 {
			// more than one hosts not allowed
			badReqResponse.Write(conn)
			shouldCloseConn = true
			break
		}

		if req.Headers.Get("content-length") != "" && req.Headers.Get("transfer-encoding") != "" {
			// requests containing both content length and transfer encoding
			// headers MAY be rejected by the server as per the RFC
			// https://datatracker.ietf.org/doc/html/rfc9112#section-6.1-15
			// we're going to reject it
			badReqResponse.Write(conn)
			shouldCloseConn = true
			break
		}

		_, err = req.TransferEncodings()
		if err != nil {
			// last transfer encoding must be chunked
			// https://datatracker.ietf.org/doc/html/rfc9112#section-6.3-2.4.3
			badReqResponse.Write(conn)
			shouldCloseConn = true
			break
		}

		resp := s.handler(req)
		resp.GetHeaders().Remove("date")
		resp.WithHeader("date", time.Now().Format(time.RFC1123))
		if shouldCloseConn {
			resp.WithHeader("connection", "close")
		}

		if respEtag, reqEtag := resp.GetHeaders().Get("etag"), req.Headers.Get("if-none-match"); respEtag != "" && reqEtag != "" {
			// response has an etag header, and
			// request has a `if-none-match` header, then
			// check both values, if match, return 304 not modified
			if respEtag == reqEtag {
				resp = response.NewBaseResponse().
					WithStatusCode(response.StatusNotModified)
			}
		}

		err = resp.Write(conn)
		if err != nil {
			log.Println("unable to write response to connection:", err)
			shouldCloseConn = true
			break
		}

		if strings.TrimSpace(strings.ToLower(req.Headers.Get("connection"))) == "close" {
			// if the client requests connection close, respect it
			shouldCloseConn = true
			break
		}

		if shouldCloseConn {
			break
		}

		// this error is already checked via the transfer encoding check
		b, _ := req.Body()
		b.Close() // discard body from buffer
	}
}

func newServer(opts ServerOpts, handler Handler) *Server {
	if opts.Recovery == nil {
		opts.Recovery = defaultRecovery
	}
	if opts.Address == "" {
		opts.Address = ":42069"
	}
	return &Server{
		opts:    opts,
		handler: handler,
	}
}

// Starts the HTTP server with the given options and handler.
func Serve(opts ServerOpts, handler Handler) (*Server, error) {
	s := newServer(opts, handler)

	errCh := make(chan error, 1)
	go func() {
		err := s.listen()
		if err != nil {
			errCh <- err
		}
	}()

	// give the server a moment to start and potentially fail
	select {
	case err := <-errCh:
		return nil, err
	case <-time.After(100 * time.Millisecond):
		return s, nil
	}
}
