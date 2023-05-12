package system_test

import (
	"context"
	"testing"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/lthibault/log"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/system"
)

func TestFSOpenResolve(t *testing.T) {
	t.Parallel()

	fs := system.FS{
		Ctx:  context.Background(),
		Log:  log.New(),
		Root: &anchor.Node{},
	}

	f, err := fs.Open(".")
	require.NoError(t, err)
	require.NotNil(t, f)

	conn := rpc.NewConn(rpc.NewStreamTransport(f.(*system.Socket)), nil)
	boot := conn.Bootstrap(context.Background())

	err = boot.Resolve(context.Background())
	require.NoError(t, err)
}
