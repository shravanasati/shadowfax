package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

func getLinesChannel(f io.ReadCloser, bufSize int, sep []byte) <-chan string {
	ch := make(chan string)
	go func ()  {
		buffer := make([]byte, bufSize)
		eof := false
		part := ""
		for !eof {
			n, err := f.Read(buffer)
			if err == io.EOF {
				eof = true
				continue
			}
			if err != nil {
				panic(err)
			}
	
			sepPos := bytes.Index(buffer[:n], sep)
			if sepPos != -1 {
				// found a newline
				part += string(buffer[:sepPos])
				ch <- part
				part = string(buffer[sepPos+len(sep) : n])
			} else {
				part += string(buffer[:n])
			}
		}
		if part != "" {
			ch <- part
		}	

		close(ch)
	}()

	return ch
}

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
		lines := getLinesChannel(conn, 8, []byte("\n"))
		for line := range lines {
			fmt.Println(line)
		}
		fmt.Println("a connection has been closed")
	}
}
