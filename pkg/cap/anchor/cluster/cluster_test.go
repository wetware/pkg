package cluster_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/ww/pkg/cap/anchor/cluster"
)

func TestIter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	view := mockView{
		{
			id:  "testid",
			ttl: time.Second * 10,
			seq: 42,
			dl:  time.Now().Add(time.Second * 10),
		},
	}

	s := cluster.ClusterServer{view}

	c := s.NewClient(nil)
	it, release := c.Iter(ctx)
	defer release()

	err := it.Next(ctx)
	assert.NoError(t, err)

	assert.NotNil(t, it.Record())
	assert.NotZero(t, it.Deadline())

	err = it.Next(ctx)
	assert.ErrorIs(t, err, cluster.ErrExhausted)
}

type iter struct {
	recs []record
	idx  int
}

func (it *iter) Next() {
	it.idx++
}

func (it iter) Record() routing.Record {
	if it.idx >= len(it.recs) {
		return nil
	}

	return it.recs[it.idx]
}

func (it iter) Deadline() time.Time {
	if it.idx >= len(it.recs) {
		return time.Time{}
	}

	return it.recs[it.idx].dl
}

func (it iter) Finish() {}

type record struct {
	id  peer.ID
	ttl time.Duration
	seq uint64
	dl  time.Time
}

func (r record) Peer() peer.ID      { return r.id }
func (r record) TTL() time.Duration { return r.ttl }
func (r record) Seq() uint64        { return r.seq }

type mockView []record

func (v mockView) Iter() routing.Iterator {
	return &iter{
		recs: v,
	}
}

func (v mockView) Lookup(id peer.ID) (routing.Record, bool) {
	for _, r := range v {
		if r.Peer() == id {
			return r, true
		}
	}

	return nil, false
}
