package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
	"github.com/shravanasati/shadowfax/internal/server"
)

const port = 42069

func main() {
	server, err := server.Serve(port, func(r *request.Request) response.Response {
		if r.RequestLine.Target == "/yourproblem" {
			return response.
				NewTextResponse("your problem is not my problem\n").
				WithStatusCode(response.StatusBadRequest)
		}

		if r.RequestLine.Target == "/myproblem" {
			return response.
				NewTextResponse("woopsie, my bad\n").
				WithStatusCode(response.StatusInternalServerError)
		}

		if r.RequestLine.Target == "/httpbin" {
			resp, err := http.Get("https://httpbin.org/stream/100")
			if err != nil {
				fmt.Printf("Error fetching httpbin: %v\n", err)
				return response.
					NewBaseResponse().
					WithStatusCode(response.StatusInternalServerError)
			}

			fmt.Printf("httpbin response status: %s\n", resp.Status)
			fmt.Printf("httpbin response headers: %v\n", resp.Header)

			sr := response.NewStreamResponse(func(w io.Writer) error {
				defer resp.Body.Close()

				buf := make([]byte, 1024)
				totalBytes := 0
				for {
					n, err := resp.Body.Read(buf)
					if err == io.EOF {
						fmt.Printf("Finished reading httpbin response, total bytes: %d\n", totalBytes)
						break
					}
					if err != nil {
						fmt.Printf("Error reading httpbin response: %v\n", err)
						return err
					}
					totalBytes += n
					fmt.Printf("Read %d bytes from httpbin, writing to client\n", n)
					w.Write(buf[:n])
				}

				return nil
			})
			return sr
		}

		if r.RequestLine.Target == "/stream" {
			sr := response.NewStreamResponse(func(w io.Writer) error {
				ticker := time.NewTicker(time.Millisecond * 100)
				defer ticker.Stop()
				deadline := time.After(2 * time.Second)

				for {
					select {
					case <-deadline:
						return nil // finishes after 2s, pipe closes
					case t := <-ticker.C:
						fmt.Fprintf(w, "%v\n", t)
					}
				}
			})

			return sr
		}

		return response.
			NewTextResponse("all good, frfr\n").
			WithStatusCode(response.StatusOK)
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
