// Package ww contains Wetware bindings for WASM guest-code.
package ww

import (
	"context"
	"unsafe"

	"capnproto.org/go/capnp/v3/rpc"

	api "github.com/wetware/ww/api/cluster"

	"github.com/stealthrocket/wazergo/types"
)

var conn = rpc.NewConn(rpc.NewStreamTransport(hostPipe{}), nil)

// Bootstrap returns the host capability exported by the Wetware
// runtime.
func Bootstrap(ctx context.Context) api.Host {
	return api.Host(conn.Bootstrap(ctx))
}

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

//go:inline
func bytesToPointer(b []byte) uint32 {
	return *(*uint32)(unsafe.Pointer(unsafe.SliceData(b)))
}
