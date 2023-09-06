//go:generate env GOOS=wasip1 GOARCH=wasm go build -o wait.wasm wait.go
package main

import (
	"context"
	"fmt"

	"github.com/wetware/pkg/guest/system"
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
	sess, err := system.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(sess)

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		fmt.Println(ctx.Err())
	// 		return
	// 	case <-time.After(1 * time.Second):
	// 		continue
	// 	}
	// }
}
