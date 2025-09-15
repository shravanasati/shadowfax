package main

import (
	"fmt"
	"net"

	"github.com/shravanasati/shadowfax/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		panic(err)
	}
	fmt.Println("listening for connections")

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("a connection has been accepted", conn.RemoteAddr().String())
		req, err := request.RequestFromReader(conn)
		if err != nil {
			fmt.Println("error parsing the request:", err)
		} else {
			fmt.Printf("Request line: \n\t- Method: %s \n\t- Target: %s \n\t- HTTP Version: %s \n", req.RequestLine.Method, req.RequestLine.Target, req.RequestLine.HTTPVersion)
			fmt.Printf("Headers: \n")
			for key, val := range req.Headers.All() {
				fmt.Printf("\t- %s: %s \n", key, val)
			}
			// fmt.Printf("Body: \n%s\n", string(req.Body))
		}
		fmt.Println("a connection has been closed")
	}
}
