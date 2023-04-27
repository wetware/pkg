package main

import (
	"context"
	"fmt"
	"os"

	ww "github.com/wetware/ww/guest/tinygo"
)

func main() {
	ctx := context.Background()
	host := ww.Bootstrap(ctx)
	err := host.Resolve(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	future, release := host.SayHi(ctx, nil)
	defer release()
	<-future.Done()
}
