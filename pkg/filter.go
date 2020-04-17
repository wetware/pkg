package ww

import (
	"container/heap"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// Filter valid heartbeat messages from stales ones.
type Filter interface {
	// Upsert attempts to insert a peer's TTL and sequence number, updating any existing
	// entries.  If Upsert returns false, the message should be discarded.
	Upsert(peer.ID, uint64, time.Duration) bool

	// Advance the filter to the specified time point.  This will expire any stale
	// entries.
	Advance(time.Time)
}

// ShardedFilter assigns peers to subfilters based on their peer.IDs in order to reduce
// mutex contention.
type ShardedFilter [256]HeapFilter

// Upsert implements Filter.Upsert.
func (f *ShardedFilter) Upsert(id peer.ID, seq uint64, ttl time.Duration) bool {
	return f[shard(id)].Upsert(id, seq, ttl)
}

// Advance implements Filter.Advance.
func (f *ShardedFilter) Advance(t time.Time) {
	for i := range f {
		f[i].Advance(t)
	}
}

func shard(id peer.ID) int {
	return int(id[len(id)-1])
}

// HeapFilter is an efficient Filter implementation that maintains a min-heap of TTLs.
type HeapFilter struct {
	o sync.Once
	sync.Mutex
	ps   map[peer.ID]*state
	heap stateHeap
}

func (f *HeapFilter) init() {
	f.ps = map[peer.ID]*state{}
	f.heap = stateHeap{}
}

// Upsert performance is roughly O(log n).
func (f *HeapFilter) Upsert(id peer.ID, seq uint64, ttl time.Duration) bool {
	f.o.Do(f.init)
	f.Lock()
	defer f.Unlock()

	state, ok := f.ps[id]
	if !ok {
		f.insertState(id, seq, ttl)
		return true
	}

	if seq > state.Seq {
		f.updateState(state, seq, ttl)
		return true
	}

	return false
}

// Advance performance is O(n), where n is the number of expired entries.
func (f *HeapFilter) Advance(t time.Time) {
	f.o.Do(f.init)
	f.Lock()
	defer f.Unlock()

	var s *state
	for {
		if s = f.heap.Peek(); s == nil || s.Deadline.After(t) {
			break
		}

		delete(f.ps, s.ID)
		heap.Pop(&f.heap) // O(log n)
	}
}

// requires locking
func (f *HeapFilter) insertState(id peer.ID, seq uint64, ttl time.Duration) {
	state := &state{
		idx:      len(f.heap),
		ID:       id,
		Seq:      seq,
		Deadline: time.Now().Add(ttl),
	}

	f.ps[id] = state
	heap.Push(&f.heap, state) // O(log n)
}

// requires locking
func (f *HeapFilter) updateState(state *state, seq uint64, ttl time.Duration) {
	state.Seq = seq
	state.Deadline = time.Now().Add(ttl)
	heap.Fix(&f.heap, state.idx) // O(log n)
}

type state struct {
	idx      int
	ID       peer.ID
	Seq      uint64
	Deadline time.Time
}

type stateHeap []*state

func (h stateHeap) Len() int           { return len(h) }
func (h stateHeap) Less(i, j int) bool { return h[i].Deadline.Before(h[j].Deadline) }
func (h stateHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]   // swap
	h[i].idx, h[j].idx = i, j // update indices
}

func (h stateHeap) Peek() *state {
	if len(h) == 0 {
		return nil
	}

	return h[0]
}

func (h *stateHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*state))
}

func (h *stateHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
