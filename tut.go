// tut
package main

import (
	"fmt"
	"os"
)

func main() {
	i := 1
	DOCUMENT_ROOT := "No"
	numproc := "1"
	fmt.Println(len(os.Args))
	for i < len(os.Args) {
		switch os.Args[i] {
		case "-r":
			i += 1
			DOCUMENT_ROOT = os.Args[i]

		case "-c":
			i += 1
			numproc = os.Args[i]
		default:
			i += 1
		}

	}
	fmt.Println("root")
	fmt.Println(DOCUMENT_ROOT)
	fmt.Println("num")
	fmt.Println(numproc)
}
