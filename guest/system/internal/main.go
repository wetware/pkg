package main

import (
	"os"

	"github.com/wetware/pkg/guest/system"
)

func main() {
	sock := system.Socket{Reader: os.Stdin}
	defer func() {
		if err := sock.Close(); err != nil {
			panic(err)
		}
	}()

	n, err := sock.Write([]byte("Hello, Go!"))
	if err != nil {
		panic(err)
	}
	if n != len("Hello, Go!") {
		panic(err)
	}
}
