package routing_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/cluster/routing"
)

var t0 = time.Date(2020, 4, 9, 8, 0, 0, 0, time.UTC)

func TestID(t *testing.T) {
	t.Parallel()

	var id = routing.ID(42)
	assert.Len(t, id.Bytes(), 8)
	assert.Len(t, id.String(), 16)

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(id)
	require.NoError(t, err, "should marshal JSON")

	var got routing.ID
	err = json.NewDecoder(&buf).Decode(&got)
	require.NoError(t, err, "should unmarshal JSON")

	assert.Equal(t, id, got)
}

func BenchmarkID(b *testing.B) {
	b.ReportAllocs()

	var id = routing.ID(42)
	b.Run("String", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.String()
		}
	})

	b.Run("Bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := id.Bytes()
			pool.Put(buf)
		}
	})
}

func TestMetaField(t *testing.T) {
	t.Parallel()

	var f routing.MetaField
	assert.Empty(t, f.String(), "should default to empty string")
}

func newPeerID() peer.ID {
	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	id, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		panic(err)
	}

	return id
}
