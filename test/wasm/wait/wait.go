//go:generate env GOOS=wasip1 GOARCH=wasm go build -o wait.wasm wait.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wetware/pkg/guest/system"
)

func main() {
	ctx := context.Background()

	sess, err := system.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Successfully bootstrapped session %v\n", sess)
	for {
		time.Sleep(1 * time.Second)
	}
}
