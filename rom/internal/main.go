package main

import (
	"fmt"
	"os"

	"github.com/wetware/pkg/guest/system"
)

func main() {
	err := system.Poll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// fmt.Printf("%x\n", stat)
	os.Exit(0)
}
