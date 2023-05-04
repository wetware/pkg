// Package ww contains Wetware bindings for WASM guest-code.
package ww

import (
	"context"
	"io/fs"
	"os"

	"capnproto.org/go/capnp/v3/rpc"

	"github.com/wetware/ww/internal/api/cluster"
	ww_fs "github.com/wetware/ww/pkg/csp/fs"
)

// Bootstrap returns the host capability exported by the Wetware
// runtime.
func Bootstrap(ctx context.Context) cluster.Host {
	f, err := os.Open("/")
	if err != nil {
		panic(err)
	}
	guestConn := fs.File(f).(ww_fs.File).Conn()
	conn := rpc.NewConn(rpc.NewStreamTransport(guestConn), nil) // TODO missing bootstrap client?

	return cluster.Host(conn.Bootstrap(ctx))
}
