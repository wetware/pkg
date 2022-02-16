package cluster

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
)

const bufSize = int32(8)

var ErrNotFound = errors.New("not found")

type Client api.Cluster

func (cl Client) Iter() *Iterator {
	return newIterator(api.Cluster(cl), bufSize)
}

func (cl Client) Lookup(ctx context.Context, peerID peer.ID) (cluster.Record, error) {
	fr, release := api.Cluster(cl).Lookup(ctx, func(r api.Cluster_lookup_Params) error {
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

func newRecords(capRecs api.Cluster_Record_List) ([]cluster.Record, error) {
	recs := make([]cluster.Record, 0, capRecs.Len())
	for i := 0; i < capRecs.Len(); i++ {
		rec, err := newRecord(capRecs.At(i))
		if err != nil {
			return nil, err
		}
		recs = append(recs, rec)
	}
	return recs, nil
}

func newRecord(capRec api.Cluster_Record) (cluster.Record, error) {
	peerID, err := capRec.Peer()
	if err != nil {
		return nil, err
	}
	return Record{
		peerID:   peer.ID(peerID),
		deadline: time.Now().Add(time.Duration(capRec.Ttl())),
		seq:      capRec.Seq(),
	}, nil
}

type Record struct {
	peerID   peer.ID
	deadline time.Time
	seq      uint64
}

func (rec Record) Peer() peer.ID {
	return rec.peerID
}

func (rec Record) TTL() time.Duration {
	return time.Until(rec.deadline)
}

func (rec Record) Seq() uint64 {
	return rec.seq
}
