// Package ww contains Wetware bindings for WASM guest-code.
package ww

import (
	"context"
	"unsafe"

	"capnproto.org/go/capnp/v3/rpc"

	"github.com/wetware/ww/internal/api/cluster"

	"github.com/stealthrocket/wazergo/types"
)

var (
	conn = rpc.NewConn(rpc.NewStreamTransport(hostPipe{}), nil)
	h    = cluster.Host(conn.Bootstrap(context.Background()))
)

type hostPipe struct{}

func (hostPipe) Read(b []byte) (int, error) {
	var n uint32
	err := hostRead(bytesToPointer(b), uint32(len(b)), &n)
	return int(n), types.Errno(err)
}

func (hostPipe) Write(b []byte) (int, error) {
	var n uint32
	err := hostWrite(bytesToPointer(b), uint32(len(b)), &n)
	return int(n), types.Errno(err)
}

func (hostPipe) Close() error {
	return types.Errno(hostClose())
}

func Boostrasp() cluster.Host {
	return h
}

// func Root(ctx context.Context) (anchor.Anchor, capnp.ReleaseFunc) {
// 	// FIXME:  return the root anchor.  Right now we just return the
// 	// local host anchor.
// 	f, release := host.Root(ctx, nil)
// 	return anchor.Anchor(f.Root()), release
// }

//go:inline
func bytesToPointer(b []byte) uint32 {
	return *(*uint32)(unsafe.Pointer(unsafe.SliceData(b)))
}
