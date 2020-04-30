package server

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

type filter interface {
	Upsert(peer.ID, uint64, time.Duration) bool
	Contains(peer.ID) bool
}

type basicFilter struct {
	sync.RWMutex
	es map[peer.ID]*entry
}

func newBasicFilter() *basicFilter {
	return &basicFilter{es: make(map[peer.ID]*entry, 32)}
}

func (f *basicFilter) Contains(id peer.ID) (found bool) {
	f.RLock()
	defer f.RUnlock()

	_, found = f.es[id]
	return
}

func (f *basicFilter) Upsert(id peer.ID, seq uint64, ttl time.Duration) (ok bool) {
	f.Lock()
	var e *entry
	if e, ok = f.es[id]; !ok {
		e = &entry{pool: f, key: id, seq: seq}
		e.timer = time.AfterFunc(ttl, e.expire)
		f.es[id] = e

		f.Unlock()
		return
	}
	f.Unlock()

	return e.update(seq, ttl)
}

type entry struct {
	pool *basicFilter

	key       peer.ID
	seq       uint64
	timer     *time.Timer
	timerLock sync.Mutex
}

func (e *entry) update(seq uint64, ttl time.Duration) bool {
	var old uint64
	for {
		if old = atomic.LoadUint64(&e.seq); seq <= old {
			return false
		}

		e.timerLock.Lock()
		defer e.timerLock.Unlock()

		// seq may have changed while we were locking
		if !atomic.CompareAndSwapUint64(&e.seq, old, seq) {
			continue
		}

		e.setTTL(ttl)
		return true
	}
}

// requires locking
func (e *entry) setTTL(d time.Duration) {
	if !e.timer.Stop() {
		e.pool.Lock()
		defer e.pool.Unlock()

		e.pool.es[e.key] = e
	}

	e.timer = time.AfterFunc(d, e.expire)
}

func (e *entry) expire() {
	e.pool.Lock()
	defer e.pool.Unlock()

	delete(e.pool.es, e.key)
}
