package socket_test

import (
	"crypto/rand"
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/wetware/pkg/api/boot"
	"github.com/wetware/pkg/boot/socket"
)

func BenchmarkRecord(b *testing.B) {
	var env *record.Envelope

	b.Run("Seal", func(b *testing.B) {
		_, seg := capnp.NewSingleSegmentMessage(nil)
		pkt, _ := boot.NewRootPacket(seg)

		pkt.SetNamespace("casm.test")
		pkt.SetResponse()
		pkt.Response().SetPeer(string(newPeerID()))

		as, _ := pkt.Response().NewAddrs(2)
		as.Set(0, ma.StringCast("/ip4/127.0.0.1/tcp/92").Bytes())
		as.Set(1, ma.StringCast("/ip6/::1/tcp/92").Bytes())

		pk, _, _ := crypto.GenerateEd25519Key(rand.Reader)

		for i := 0; i < b.N; i++ {
			env, _ = record.Seal((*socket.Record)(&pkt), pk)
		}
	})

	b.Run("Unseal", func(b *testing.B) {
		var rec socket.Record
		for i := 0; i < b.N; i++ {
			_ = env.TypedRecord(&rec)
		}
	})

	// Log size
	bs, _ := env.Marshal()
	b.Logf("payload size:  %d bytes", len(bs))
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
