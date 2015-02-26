// httpd
package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	port := ":7777"
	address, err := net.ResolveTCPAddr("127.0.0.1", port)
	checkError(err)
	listener, err := net.ListenTCP("tcp", address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)

	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	var buf [512]byte
	n, err := conn.Read(buf[0:])

	for err2 == nil && err == nil {
		n, err := conn.Read(buf[0:])
		if err != nil {
			return
		}
		fmt.Println(string(buf[0:]))
		fmt.Println(int(n))

		_, _ = conn.Write([]byte("GET / HTTP/1.1 \r\n"))
		_, _ = conn.Write([]byte("Host: 127.0.0.1 \r\n"))
		_, _ = conn.Write([]byte("\r\n"))
		_, err2 := conn.Write(buf[0:])
	}
}
