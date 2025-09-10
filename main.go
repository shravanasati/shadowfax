package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
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
	f, err := os.Open("messages.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	ch := getLinesChannel(f)
	for item := range ch {
		fmt.Printf("read: %s\n", item)
	}
}
