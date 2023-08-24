package ww

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"

	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/system"
)

const (
	Version = "0.1.0"
)

type BootFunc[T ~capnp.ClientKind] func(T) *rpc.Options

// Ww is the execution context for WebAssembly (WASM) bytecode,
// allowing it to interact with (1) the local host and (2) the
// cluster environment.
type Ww[T ~capnp.ClientKind] struct {
	NS      string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Client  T
	Options BootFunc[T]
}

// String returns the cluster namespace in which the wetware is
// executing. If ww.NS has been assigned a non-empty string, it
// returns the string unchanged.  Else, it defaults to "ww".
func (ww *Ww[T]) String() string {
	if ww.NS != "" {
		return ww.NS
	}

	return "ww"
}

// Exec compiles and runs the ww instance's ROM in a WASM runtime.
// It returns any error produced by the compilation or execution of
// the ROM.
func (ww Ww[T]) Exec(ctx context.Context, rom rom.ROM) error {
	sock := &system.Socket{
		NS:  ww.NS,
		ROM: rom,
		ID:  nextRoutingID(),
	}

	opts := &rpc.Options{
		BootstrapClient: capnp.Client(ww.Client),
		ErrorReporter: system.ErrorReporter{
			Logger: sock.Logger(),
		},
	}

	system, err := sock.Bind(ctx)
	if err != nil {
		return fmt.Errorf("socket: bind: %w", err)
	}
	defer system.Close(ctx)

	conn := rpc.NewConn(sock, opts)
	defer conn.Close()

	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func nextRoutingID() (r routing.ID) {
	if err := binary.Read(rand.Reader, binary.LittleEndian, &r); err != nil {
		panic(err)
	}

	return
}
