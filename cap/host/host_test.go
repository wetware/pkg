package host_test

import (
	"context"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
)

func TestHost_login(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("EmptySession", func(t *testing.T) {
		t.Parallel()

		policy := host.AuthDisabled(host.Session{
			// Don't pass any capabilities
		})

		host := host.Host(policy.Client())
		defer host.Release()

		sess, err := host.Login(context.Background(), api.Signer{})
		require.NoError(t, err, "login should succeed")

		// ensure all clients are null
		for name, c := range map[string]capnp.Client{
			"view":   capnp.Client(sess.View),
			"pubsub": capnp.Client(sess.Router),
			// "foo": capnp.Client(sess.Foo),
		} {
			assert.Equal(t, capnp.Client{}, c, "%s should be null", name)
		}
	})

	t.Run("ProvideView", func(t *testing.T) {
		t.Parallel()

		rt := mockRoutingTable{}
		vs := view.Server{RoutingTable: rt}

		policy := host.AuthDisabled(host.Session{
			View: vs.View(),
		})

		host := host.Host(policy.Client())
		defer host.Release()

		sess, err := host.Login(context.Background(), api.Signer{})
		require.NoError(t, err, "login should succeed")

		all := view.NewQuery(view.All())
		f, release := sess.View.Lookup(context.Background(), all)
		defer release()

		r, err := f.Await(context.Background())
		require.NoError(t, err, "should resolve record")
		require.NotNil(t, r, "record should not be nil")
	})
}

type mockRoutingTable struct{}

func (mockRoutingTable) Snapshot() routing.Snapshot {
	return mockSnapshot{}
}

type mockSnapshot struct{}

func (mockSnapshot) Get(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}
func (mockSnapshot) GetReverse(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}
func (mockSnapshot) LowerBound(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}
func (mockSnapshot) ReverseLowerBound(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}

type mockIterator []mockRecord

func (it *mockIterator) Next() routing.Record {
	switch len(*it) {
	case 0:
		return nil

	case 1:
		r := (*it)[0]
		*it = nil
		return r

	default:
		r := (*it)[0]
		*it = (*it)[1:]
		return r
	}
}

type mockRecord struct{}

func (mockRecord) Peer() peer.ID               { return "foobar" }
func (mockRecord) Server() routing.ID          { return routing.ID(42) }
func (mockRecord) TTL() time.Duration          { return time.Second }
func (mockRecord) Seq() uint64                 { return 1 }
func (mockRecord) Host() (string, error)       { return "foo", nil }
func (mockRecord) Meta() (routing.Meta, error) { return routing.Meta{}, nil }
