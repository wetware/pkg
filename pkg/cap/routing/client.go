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

func (rt RoutingClient) Iter(ctx context.Context) *IteratorV2 {
	return newIterator(ctx, rt.rt, 1)
}

// Lookup
func (rt RoutingClient) Lookup(ctx context.Context, peerID peer.ID) (cluster.Record, bool) {
	fr, release := rt.rt.Lookup(ctx, func(r api.Routing_lookup_Params) error {
		return r.SetPeerID(string(peerID))
	})
	defer release()

	rec, ok, _ := FutureLookup{fr}.Struct()
	return rec, ok
}

type FutureLookup struct {
	fl api.Routing_lookup_Results_Future
}

func (fl FutureLookup) Struct() (cluster.Record, bool, error) {
	s, err := fl.fl.Struct()
	if err != nil {
		return nil, false, err
	}
	rec, err := s.Record()
	if err != nil {
		return nil, s.Ok(), nil
	}
	return newRecord(rec), s.Ok(), nil
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
