package routing

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/routing"
)

type RoutingClient struct {
	rt api.Routing
}

func (rt RoutingClient) Iter(ctx context.Context, bufSize int32) *Iterator {
	return newIterator(ctx, rt.rt, bufSize)
}

func (rt RoutingClient) Lookup(ctx context.Context, peerID peer.ID) (cluster.Record, bool) {
	fr, release := rt.rt.Lookup(ctx, func(r api.Routing_lookup_Params) error {
		return r.SetPeerID(string(peerID))
	})
	defer release()

	s, err := fr.Struct()
	if err != nil {
		return nil, false
	}

	rec, err := s.Record()
	if err != nil {
		return nil, false
	}
	return newRecord(rec), s.Ok()
}

func newRecord(capRec api.Record) cluster.Record {
	peerID, err := capRec.Peer()
	if err != nil {
		peerID = ""
	}
	return Record{
		peerID: peer.ID(peerID),
		ttl:    time.Duration(capRec.Ttl()),
		seq:    capRec.Seq(),
	}
}

type Record struct {
	peerID peer.ID
	ttl    time.Duration
	seq    uint64
}

func (rec Record) Peer() peer.ID {
	return rec.peerID
}

func (rec Record) TTL() time.Duration {
	return rec.ttl
}

func (rec Record) Seq() uint64 {
	return rec.seq
}
