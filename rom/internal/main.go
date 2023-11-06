package main

import (
	"fmt"
	"os"

	"github.com/wetware/pkg/guest/system"
)

func main() {
	sock := system.Socket{Reader: os.Stdin}
	defer sock.Close()

	fmt.Println("Hello, Wetware!")
	fmt.Println("ROM: ", os.Args[0])
}
