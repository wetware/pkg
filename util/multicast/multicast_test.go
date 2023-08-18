package multicast_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/util/multicast"
	"golang.org/x/sync/errgroup"
)

func TestMulticast(t *testing.T) {
	t.Parallel()

	group, err := net.ResolveUDPAddr("udp", "228.8.8.8:8822")
	require.NoError(t, err)

	sock, err := multicast.Bind(group)
	require.NoError(t, err)
	defer sock.Close()

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		for i := 0; i < 10; i++ {
			err := sock.Send(ctx, []byte("hello, wetware!"))
			require.NoError(t, err)
			time.Sleep(time.Microsecond * 100)
		}

		return nil
	})
	g.Go(func() error {
		buf, err := sock.Recv(ctx)
		if err != nil {
			return err
		}

		require.Equal(t, "hello, wetware!", string(buf))
		t.Log(string(buf))
		return nil
	})

	assert.NoError(t, g.Wait(), "multicast failure")
}
