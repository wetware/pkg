package main

import (
	"context"
	"fmt"
	"os"

	ww "github.com/wetware/ww/guest/tinygo"
)

func main() {
	ctx := context.Background()
	err := ww.Bootstrap(ctx).Resolve(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
