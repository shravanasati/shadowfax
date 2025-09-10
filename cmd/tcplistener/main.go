package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	ch := make(chan string)
	go func ()  {
		buffer := make([]byte, 8)
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
	
			newlinePos := bytes.IndexByte(buffer[:n], '\n')
			if newlinePos != -1 {
				// found a newline
				part += string(buffer[:newlinePos])
				ch <- part
				part = string(buffer[newlinePos+1 : n])
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
		lines := getLinesChannel(conn)
		for line := range lines {
			fmt.Println(line)
		}
		fmt.Println("a connection has been closed")
	}
}
