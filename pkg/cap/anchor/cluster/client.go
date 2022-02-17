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

type Client api.Cluster

func (cl Client) Iter(ctx context.Context) (*Iterator, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	h := make(handler, defaultMaxInflight)

	it, release := newIterator(ctx, api.Cluster(cl), h)
	return it, func() {
		cancel()
		release()
	}
}

func (cl Client) Lookup(ctx context.Context, peerID peer.ID) (FutureRecord, capnp.ReleaseFunc) {
	f, release := api.Cluster(cl).Lookup(ctx, func(r api.Cluster_lookup_Params) error {
		return r.SetPeerID(string(peerID))
	})
	return FutureRecord(f), release
}

type FutureRecord api.Cluster_lookup_Results_Future

func (f FutureRecord) Struct() (Record, error) {
	res, err := api.Cluster_lookup_Results_Future(f).Struct()
	if err != nil {
		return Record{}, err
	}

	rec, err := res.Record()
	return Record(rec), err
}

type Record api.Cluster_Record

func (rec Record) Peer() (peer.ID, error) {
	s, err := api.Cluster_Record(rec).Peer()
	if err != nil {
		return "", err
	}

	return peer.IDFromString(s)
}

func (rec Record) TTL() time.Duration {
	return time.Duration(api.Cluster_Record(rec).Ttl())
}

func (rec Record) Seq() uint64 {
	return api.Cluster_Record(rec).Seq()
}
