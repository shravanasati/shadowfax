package main

import (
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/shravanasati/shadowfax/internal/server"
)

const port = 42069

func main() {
	server, err := server.Serve(port, func(w io.Writer, r *request.Request) *server.HandlerError {
		if r.RequestLine.Target == "/yourproblem" {
			return server.NewHandlerError(response.StatusBadRequest, "your problem is not my problem\n")
		}
		if r.RequestLine.Target  == "/myproblem" {
			return server.NewHandlerError(response.StatusInternalServerError, "woopsie, my bad\n")
		}
		w.Write([]byte("all good, frfr\n"))
		return nil
	})
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
