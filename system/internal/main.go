//go:generate tinygo build -o main.wasm -target=wasi -scheduler=asyncify main.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/ww/pkg/anchor"
)

func main() {
	f, err := os.OpenFile("ww", os.O_RDWR, 0)
	if err != nil {
		panic(fmt.Errorf("failed to open path: %w", err))
	}
	defer f.Close()

	conn := rpc.NewConn(rpc.NewStreamTransport(f), &rpc.Options{
		ErrorReporter: errLogger{},
	})
	defer conn.Close()

	buf := make([]byte, 1024<<1)
	n := runtime.Stack(buf, true)
	buf = buf[:n]
	fmt.Println(n, string(buf))

	client := conn.Bootstrap(context.Background())
	root := anchor.Anchor(client)
	defer root.Release()

	fmt.Println("resolving")
	if err := client.Resolve(context.TODO()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("blocking until cancel")
	<-conn.Done()
}

type errLogger struct{}

func (e errLogger) ReportError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
