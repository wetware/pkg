//go:generate env GOOS=wasip1 GOARCH=wasm go build -o bootstrap.wasm bootstrap.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wetware/pkg/guest/system"
)

func main() {
	ctx := context.Background()

	caps, err := system.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}

	if len(caps) == 0 {
		fmt.Println("No capabilities found in bootstrap")
		os.Exit(1)
	}

	session, err := system.Login(ctx, caps[0])
	if err != nil {
		panic(err)
	}

	fmt.Printf("Successfully bootstrapped session %v\n", session)
}
