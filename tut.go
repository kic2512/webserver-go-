// tut
package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main() {
	s := "GET / HTTP/1.1\r\nHost: 127.0.0.1 Content-Type: application/octet-stream\r\n\r\n"
	result := strings.Split(s, ' ')
	fmt.Printf(result)
}
