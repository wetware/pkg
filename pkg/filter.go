package ww

import (
	"container/heap"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// filter identifies stale heartbeats.
type filter struct {
	sync.Mutex
	ttl  time.Duration
	ps   map[peer.ID]*state
	heap stateHeap
}

// newFilter .
func newFilter(ttl time.Duration) *filter {
	return &filter{
		ps:   map[peer.ID]*state{},
		heap: stateHeap{},
	}
}

// Upsert a sequence number for a given peer if it is not stale.  Returns false if the
// sequence number, seq, is stale.
func (f *filter) Upsert(id peer.ID, seq uint64) bool {
	f.Lock()
	defer f.Unlock()

	state, ok := f.ps[id]
	if !ok {
		f.insertState(id, seq)
		return true
	}

	if seq > state.Seq {
		f.updateState(state, seq)
		return true
	}

	return false
}

// Advance the filter to the specified time point.  This will expire any stale entries.
func (f *filter) Advance(t time.Time) {
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
func (f *filter) insertState(id peer.ID, seq uint64) {
	state := &state{
		idx:      len(f.heap),
		ID:       id,
		Seq:      seq,
		Deadline: time.Now().Add(f.ttl),
	}

	f.ps[id] = state
	heap.Push(&f.heap, state) // O(log n)
}

// requires locking
func (f *filter) updateState(state *state, seq uint64) {
	state.Seq = seq
	state.Deadline = time.Now().Add(f.ttl)
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
