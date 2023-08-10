package socket

import (
	"errors"

	"capnproto.org/go/capnp/v3"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/wetware/pkg/api/boot"
)

// ErrNoListenAddrs is returned if the host has not exported any listen addresses.
var ErrNoListenAddrs = errors.New("no listen addrs")

type Host interface {
	ID() peer.ID
	Addrs() []ma.Multiaddr
}

// Sealer is a higher-order function capable of sealing a
// record for a specific peer. It prevents the cache from
// having to manage private keys.
type Sealer func(record.Record) (*record.Envelope, error)

// RecordCache is a thread-safe, fixed-size cache that tracks both
// least-frequently-used and most-recently-accessed records. It is
// used to amortize the cost of signing discovery packets.
type RecordCache lru.TwoQueueCache[key, *record.Envelope]

// NewCache creates a new RecordCache with the given size.
// The cache uses twin queues algorithm that tracks both
// recently-added entries and recently-evicted ("ghost")
// entries, in order to reduce churn.
//
// The cache MUST have a maximum size of at least 2.  If
// size < 2, a default size of 8 is used.
func NewCache(size int) *RecordCache {
	if size < 2 {
		size = 8
	}

	c, err := lru.New2Q[key, *record.Envelope](size)
	if err != nil {
		panic(err)
	}

	return (*RecordCache)(c)
}

// Reset invalidates previous entries.
func (c *RecordCache) Reset() { c.cache().Purge() }

// LoadRequest searches the cache for a signed request packet for ns
// and returns it, if found. Else, it creates and signs a new packet
// and adds it to the cache.
func (c *RecordCache) LoadRequest(seal Sealer, id peer.ID, ns string) (*record.Envelope, error) {
	if e, ok := c.cache().Get(requestKey(ns)); ok {
		return e, nil
	}

	return c.storeCache(request(id), seal, requestKey(ns))
}

// LoadSurveyRequest searches the cache for a signed survey packet
// with distance 'dist', and returns it if found. Else, it creates
// and signs a new survey-request packet and adds it to the cache.
func (c *RecordCache) LoadSurveyRequest(seal Sealer, id peer.ID, ns string, dist uint8) (*record.Envelope, error) {
	if e, ok := c.cache().Get(surveyKey(ns, dist)); ok {
		return e, nil
	}

	return c.storeCache(surveyRequest(id, dist), seal, surveyKey(ns, dist))
}

// LoadResponse searches the cache for a signed response packet for ns
// and returns it, if found. Else, it creates and signs a new response
// packet and adds it to the cache.
func (c *RecordCache) LoadResponse(seal Sealer, h Host, ns string) (*record.Envelope, error) {
	if e, ok := c.cache().Get(responseKey(ns)); ok {
		return e, nil
	}

	return c.storeCache(response(h), seal, responseKey(ns))
}

func (c *RecordCache) storeCache(bind bindFunc, seal Sealer, key key) (e *record.Envelope, err error) {
	if e, err = newCacheEntry(bind, seal, key); err == nil {
		c.cache().Add(key, e)
	}

	return
}

func (c *RecordCache) cache() *lru.TwoQueueCache[key, *record.Envelope] {
	return (*lru.TwoQueueCache[key, *record.Envelope])(c)
}

type bindFunc func(boot.Packet) error

type key struct {
	ns   string
	dist uint8
	kind keyKind
}

func requestKey(ns string) key {
	return key{ns: ns, kind: reqKind}
}

func surveyKey(ns string, dist uint8) key {
	return key{ns: ns, dist: dist, kind: survReqKind}
}

func responseKey(ns string) key {
	return key{ns: ns, kind: resKind}
}

type keyKind uint8

const (
	reqKind keyKind = iota
	resKind
	survReqKind
)

func newCacheEntry(bind bindFunc, seal Sealer, key key) (*record.Envelope, error) {
	p, err := newPacket(capnp.SingleSegment(nil), key.ns)
	if err != nil {
		return nil, err
	}

	if err = bind(p); err != nil {
		return nil, err
	}

	return seal((*Record)(&p))
}

func request(from peer.ID) func(boot.Packet) error {
	return func(p boot.Packet) error {
		p.SetRequest()
		return p.Request().SetFrom(string(from))
	}
}

func surveyRequest(from peer.ID, dist uint8) bindFunc {
	return func(p boot.Packet) error {
		p.SetSurvey()
		p.Survey().SetDistance(dist)
		return p.Survey().SetFrom(string(from))
	}
}

func response(h Host) bindFunc {
	return func(p boot.Packet) error {
		p.SetResponse()

		if err := p.Response().SetPeer(string(h.ID())); err != nil {
			return err
		}

		as, err := p.Response().NewAddrs(int32(len(h.Addrs())))
		if err == nil {
			for i, addr := range h.Addrs() {
				if err = as.Set(i, addr.Bytes()); err != nil {
					break
				}
			}
		}

		return err
	}
}

func newPacket(a capnp.Arena, ns string) (boot.Packet, error) {
	_, s, err := capnp.NewMessage(a)
	if err != nil {
		return boot.Packet{}, err
	}

	p, err := boot.NewRootPacket(s)
	if err != nil {
		return boot.Packet{}, err
	}

	return p, p.SetNamespace(ns)
}
