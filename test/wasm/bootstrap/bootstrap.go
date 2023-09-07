//go:generate env GOOS=wasip1 GOARCH=wasm go build -o bootstrap.wasm bootstrap.go
package main

import (
	"context"
	"fmt"

	"github.com/wetware/pkg/guest/system"
)

func main() {
	ctx := context.Background()

	sess, err := system.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Successfully bootstrapped session %v\n", sess)
}
