//go:generate env GOOS=wasip1 GOARCH=wasm gotip build -o explorehost.wasm explorehost.go

/*
 * Run with:
 *  - `ww start`  # to create a cluster node
 *  - `ww cluster run explorehost.wasm`. It will pass a host capability by default to the wasm guest
 */

package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/ww/api/cluster"
)

func main() {
	if err := doRpc(); err != nil {
		panic(err)
	}
}

func doRpc() error {
	fd := 3 // pre-opened tcp conn listener
	f := os.NewFile(uintptr(fd), "")

	if err := syscall.SetNonblock(fd, false); err != nil {
		return err
	}

	defer f.Close()

	l, err := net.FileListener(f)
	if err != nil {
		return err
	}
	defer l.Close()

	tcpConn, err := l.Accept()
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		ErrorReporter: errLogger{},
	})
	defer conn.Close()

	client := conn.Bootstrap(context.Background())
	host := cluster.Host(client)
	defer host.Release()

	if err := client.Resolve(context.Background()); err != nil {
		log.Fatal(err)
	}

	if !host.IsValid() {
		return errors.New("invalid host")
	}

	fmt.Println("Success")
	return nil
}

type errLogger struct{}

func (e errLogger) ReportError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
