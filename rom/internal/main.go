package main

import (
	"fmt"
	"os"

	"github.com/wetware/pkg/guest/system"
)

func main() {
	status := system.Poll()
	fmt.Printf("%0b\n", status)
	os.Exit(int(status))
}
