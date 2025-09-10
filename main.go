package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	f, err := os.Open("messages.txt")
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 8)
	eof := false
	for !eof {
		_, err := f.Read(buffer)
		if err == io.EOF {
			eof = true
			continue
		}
		if err != nil {
			panic(err)
		}
		fmt.Printf("read: %s\n", buffer)
	}
}