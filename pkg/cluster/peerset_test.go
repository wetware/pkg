package cluster

import (
	"math/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
)

var t0 = time.Date(2020, 4, 9, 8, 0, 0, 0, time.UTC)

func TestFilter(t *testing.T) {

	id := randID()
	f := new(filter)

	assert.NotPanics(t, func() { f.Advance(t0) },
		"advancing an empty filter should not panic.")

	assert.False(t, f.Contains(id),
		"canary failed:  ID should not be present in empty filter.")

	assert.True(t, f.Upsert(id, 0, time.Second),
		"upsert of new ID should succeed.")

	assert.True(t, f.Contains(id),
		"filter should contain ID %s after INSERT.", id)

	assert.True(t, f.Upsert(id, 3, time.Second),
		"upsert of existing ID with higher sequence number should succeed.")

	assert.True(t, f.Contains(id),
		"filter should contain ID %s after UPDATE", id)

	assert.False(t, f.Upsert(id, 1, time.Second),
		"upsert of existing ID with lower sequence number should fail.")

	assert.True(t, f.Contains(id),
		"filter should contain ID %s after FAILED UPDATE.", id)

	assert.Contains(t, f.Peers(), id,
		"ID should appear in peer.IDSlice")

	f.Advance(t0.Add(time.Millisecond * 100))
	assert.True(t, f.Contains(id),
		"advancing by less than the TTL amount should NOT cause eviction.")

	f.Advance(t0.Add(time.Second))
	assert.False(t, f.Contains(id),
		"advancing by more than the TTL amount should cause eviction")
}

func randID() peer.ID {
	return newID(randStr(5))
}

func randStr(n int) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	b := make([]rune, n)
	for i := range b {
		b[i] = rune(alphabet[rand.Intn(len(alphabet))])
	}

	return string(b)
}

func hash(b []byte) []byte {
	h, _ := multihash.Sum(b, multihash.SHA2_256, -1)
	return []byte(h)
}

func newID(s string) peer.ID {
	id, err := peer.IDB58Decode(base58.Encode(hash([]byte(s))))
	if err != nil {
		panic(err)
	}

	return id
}
