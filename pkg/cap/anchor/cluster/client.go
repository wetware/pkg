package cluster

import (
	"context"
	"errors"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-core/peer"
	api "github.com/wetware/ww/internal/api/cluster"
)

var ErrNotFound = errors.New("not found")

type View api.View

func (cl View) Iter(ctx context.Context) (*Iterator, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	h := make(handler, defaultMaxInflight)

	it, release := newIterator(ctx, api.View(cl), h)
	return it, func() {
		cancel()
		release()
	}
}

func (cl View) Lookup(ctx context.Context, peerID peer.ID) (FutureRecord, capnp.ReleaseFunc) {
	f, release := api.View(cl).Lookup(ctx, func(r api.View_lookup_Params) error {
		return r.SetPeerID(string(peerID))
	})
	return FutureRecord(f), release
}

type FutureRecord api.View_lookup_Results_Future

func (f FutureRecord) Struct() (Record, error) {
	res, err := api.View_lookup_Results_Future(f).Struct()
	if err != nil {
		return Record{}, err
	}

	rec, err := res.Record()
	return Record(rec), err
}

type Record api.View_Record

func (rec Record) Peer() (peer.ID, error) {
	s, err := api.View_Record(rec).Peer()
	if err != nil {
		return "", err
	}

	return peer.IDFromString(s)
}

func (rec Record) TTL() time.Duration {
	return time.Duration(api.View_Record(rec).Ttl())
}

func (rec Record) Seq() uint64 {
	return api.View_Record(rec).Seq()
}
