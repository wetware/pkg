package cluster

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
)

var ErrNotFound = errors.New("not found")

type ClusterClient struct {
	cl api.Cluster
}

func (cl ClusterClient) Iter(bufSize int32) *Iterator {
	return newIterator(cl.cl, bufSize)
}

func (cl ClusterClient) Lookup(ctx context.Context, peerID peer.ID) (cluster.Record, error) {
	fr, release := cl.cl.Lookup(ctx, func(r api.Cluster_lookup_Params) error {
		return r.SetPeerID(string(peerID))
	})
	defer release()

	s, err := fr.Struct()
	if err != nil {
		return nil, err
	}

	if !s.Ok() {
		return nil, nil
	}

	rec, err := s.Record()
	if err != nil {
		return nil, err
	}
	return newRecord(rec)
}

func newRecord(capRec api.Record) (cluster.Record, error) {
	peerID, err := capRec.Peer()
	if err != nil {
		return nil, err
	}
	return Record{
		peerID: peer.ID(peerID),
		ttl:    time.Duration(capRec.Ttl()),
		seq:    capRec.Seq(),
	}, nil
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
