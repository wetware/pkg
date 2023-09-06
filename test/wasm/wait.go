//go:generate env GOOS=wasip1 GOARCH=wasm go build -o wait.wasm wait.go
package main

import (
	"context"
	"fmt"
	"time"

	ww "github.com/wetware/pkg/guest/system"
)

func main() {
	ctx := context.Background()

	// if false {
	// _, err := ww.Init(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	// defer func() {
	// 	for _, cap := range self.Caps {
	// 		cap.Release()
	// 	}
	// }()
	// }
	s, err := ww.Bootstrap(ctx)

	if err != nil {
		panic(err)
	}
	fmt.Println(s)

	for {
		select {
		case <-ctx.Done():
			fmt.Println(ctx.Err())
			return
		case <-time.After(1 * time.Second):
			continue
		}
	}
}
