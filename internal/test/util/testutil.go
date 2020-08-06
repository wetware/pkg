package testutil

import (
	"math/rand"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multihash"
)

// RandID creates a random peer.ID
func RandID() peer.ID {
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
