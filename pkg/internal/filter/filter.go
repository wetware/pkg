package filter

import (
	"sync"
	"time"
	"unsafe"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/treap"
)

var handle = treap.Handle{
	CompareKeys:    pidComparator,
	CompareWeights: treap.TimeComparator,
}

// Filter tracks the liveliness of hosts in a cluster.
type Filter interface {
	Peers() peer.IDSlice
	Contains(peer.ID) bool
	Advance(time.Time)
	Upsert(peer.ID, uint64, time.Duration) bool
}

// New filter
func New() Filter {
	return &filter{}
}

// TODO(performance): replace lock-based CAS with a proper sync/atomic CAS loop.
// See `atomic.Value` for implementation hints (especially wrt unsafe.Pointer magic).
type filter struct {
	mu sync.RWMutex

	t time.Time
	n *treap.Node
}

func (f *filter) Peers() (ids peer.IDSlice) {
	ids = make(peer.IDSlice, 0, 128)

	for it := handle.Iter(f.root()); it.Next(); {
		ids = append(ids, it.Key.(peer.ID))
	}

	return
}

func (f *filter) Contains(id peer.ID) (ok bool) {
	_, ok = handle.Get(f.root(), id)
	return
}

func (f *filter) Advance(t time.Time) {
	// nop if t <= f.t
	if !f.advance(t) {
		return
	}

	// CAS loop
	var old, new *treap.Node
	for {
		if old = f.root(); old == nil { // atomic
			break
		}

		// pop expired entries -- does not block concurrent goroutines.
		for new = old; expired(t, new); {
			new = merge(new)
		}

		if f.cas(old, new) { // atomic
			break
		}
	}
}

func (f *filter) Upsert(id peer.ID, seq uint64, ttl time.Duration) bool {
	var ok, created bool

	for {
		old := f.root() // atomic
		new := old

		// upsert if seq is greater than the value stored in the treap -- non-blocking.
		new, created = handle.UpsertIf(new, id, seq, f.t0().Add(ttl), func(n *treap.Node) bool {
			ok = n.Value.(uint64) < seq // panics?  If so, test for nil.
			return ok
		})

		if f.cas(old, new) { // atomic
			break
		}
	}

	// The message should be processed iff the incoming message's sequence number is
	// greater than the one in the treap (ok==true) OR the id was just inserted into the
	// treap (created==true).
	return ok || created
}

func (f *filter) advance(t time.Time) (ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if ok = t.After(f.t); ok {
		f.t = t
	}

	return
}

func (f *filter) t0() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.t
}

func (f *filter) root() *treap.Node {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.n
}

func (f *filter) cas(old, new *treap.Node) (ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if ok = old == f.n; ok {
		f.n = new
	}

	return
}

func merge(n *treap.Node) *treap.Node {
	return handle.Merge(n.Left, n.Right)
}

func expired(t time.Time, n *treap.Node) (ok bool) {
	if n != nil {
		ok = handle.CompareWeights(n.Weight, t) <= 0
	}

	return
}

func pidComparator(a, b interface{}) int {
	switch {
	case a == nil:
		return -1 // N.B.:  treap is a min-heap by default
	case b == nil:
		return 1
	}

	aAsserted := a.(peer.ID)
	bAsserted := b.(peer.ID)

	return treap.StringComparator(
		*(*string)(unsafe.Pointer(&aAsserted)),
		*(*string)(unsafe.Pointer(&bAsserted)),
	)
}
