// Command slowserver is a minimal TCP server that accepts connections and never
// writes a response. It exists so you can reproduce, on demand, the "client hangs
// forever waiting for a server that never replies" scenario from the request-anatomy
// practice, without depending on a flaky external host.
//
// Usage:
//
//	go run . [addr]
//
// addr defaults to "127.0.0.1:9009". Connect to it with curl, telnet, or the
// client package in ../client, and observe that nothing ever comes back until
// your client-side timeout fires.
package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	addr := "127.0.0.1:9009"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("slowserver: listen on %s: %v", addr, err)
	}
	defer listener.Close()

	fmt.Printf("slowserver: listening on %s (every connection is accepted and then ignored forever)\n", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("slowserver: accept error: %v", err)
			continue
		}
		fmt.Printf("slowserver: accepted connection from %s, will never respond\n", conn.RemoteAddr())
		// Deliberately never read from or write to conn, and never close it here.
		// The connection stays open and silent until the client gives up.
	}
}
