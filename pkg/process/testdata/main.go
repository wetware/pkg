package main

import (
	"context"
	"os"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/ww/pkg/host"
)

/*
	build with:  tinygo build -o pkg/process/testdata/main.wasm -target=wasi -scheduler=none pkg/process/testdata/main.go
*/

//export run
func run() {
	os.Exit(99)
}

func main() {
	conn := rpc.NewConn(newWASMGuestTransport(), nil)

	host := host.Host(conn.Bootstrap(context.TODO()))

	ps := host.PubSub() // this is a syscall
}

func newWASMGuestTransport() rpc.Transport {
	return rpc.NewStreamTransport(guestMemStream{})
}

type guestMemStream struct{}

// read
// write
// close
