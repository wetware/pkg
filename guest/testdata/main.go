package main

import (
	"context"
	"time"

	ww "github.com/wetware/ww/guest"
)

func main() {
	ctx := context.Background()
	// f, err := os.OpenFile("/", os.O_RDWR, os.ModeSocket)
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()
	ww.Bootstrap(ctx)
	time.Sleep(3 * time.Second)
	// err := host.Resolve(ctx)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	// future, release := host.SayHi(ctx, nil)
	// defer release()
	// <-future.Done()
}
