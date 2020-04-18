package client

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	b58 "github.com/mr-tron/base58/base58"
	mh "github.com/multiformats/go-multihash"

	"github.com/stretchr/testify/assert"
)

func hash(b []byte) []byte {
	h, _ := mh.Sum(b, mh.SHA2_256, -1)
	return []byte(h)
}

func newID(s string) peer.ID {
	return peer.ID(b58.Encode(hash([]byte(s))))
}

func TestHeapFilter(t *testing.T) {
	const ttl = time.Second

	var f heapFilter
	id := newID("foo")
	t0 := time.Now()

	t.Run("Upsert", func(t *testing.T) {
		assert.True(t, f.Upsert(id, 1, ttl), "upserting new id should succeed")
		assert.False(t, f.Upsert(id, 1, ttl), "upserting with same sequence number should fail")
		assert.False(t, f.Upsert(id, 0, ttl), "upserting with smaller sequence number should fail")
		assert.True(t, f.Upsert(id, 2, ttl), "upserting with bigger sequence number should succeed")
	})

	t.Run("Advance", func(t *testing.T) {
		f.Advance(t0.Add(ttl * 10))
		assert.True(t, f.Upsert(id, 1, ttl), "upserting new id should succeed")
	})
}
