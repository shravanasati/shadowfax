package main

import "net"

func main() {
	addr, err := net.ResolveUDPAddr("udp4", "localhost:42069")
	if err != nil {
		panic(err)
	}

	net.DialUDP("udp4", addr, addr)
}