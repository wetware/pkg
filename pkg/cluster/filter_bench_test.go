package cluster

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	concurrency = 1024
	alphabet    = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
)

var discardBool bool

var (
	ps [128]peer.ID
)

func init() {
	for i := range ps {
		ps[i] = newID(randStr(56))
	}

}

func BenchmarkBasicFilterUpsert(b *testing.B) {
	f := newBasicFilter()
	es := eventStream(b.N)

	var wg, ready sync.WaitGroup
	wg.Add(concurrency)
	ready.Add(concurrency)
	start := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			ready.Done()
			<-start

			for _, e := range es {
				discardBool = f.Upsert(e.ID, e.Seq, e.TTL)
			}
		}()
	}

	b.ReportAllocs()
	b.ResetTimer()
	ready.Wait()
	close(start)
	wg.Wait()
}

type evt struct {
	ID  peer.ID
	Seq uint64
	TTL time.Duration
}

func eventStream(n int) []evt {
	es := make([]evt, n)
	for i := range es {
		es[i].ID = ps[i%(len(ps)-1)]
		es[i].Seq = uint64(i)
		es[i].TTL = time.Second * 10
	}

	percnt := n / 20 // 5% out-of-order sequence numbers
	for i := 0; i < percnt; i++ {
		a := rand.Intn(n - 1)
		b := rand.Intn(n - 1)

		es[a], es[b] = es[b], es[a]
	}

	return es
}

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = rune(alphabet[rand.Intn(len(alphabet))])
	}
	return string(b)
}
