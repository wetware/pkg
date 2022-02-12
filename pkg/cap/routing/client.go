package routing

import (
	"context"
	"time"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-core/peer"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/routing"
)

type RoutingClient struct {
	rt api.Routing
}

func (rt RoutingClient) Iter(ctx context.Context) (*IterHandler) {
	iterator := &IterHandler{}
	_, iterator.release = rt.rt.Iter(ctx, func(r api.Routing_iter_Params) error {
		r.SetHandler(api.Routing_Handler_ServerToClient(iterator, &defaultPolicy))
		return nil
	})
	return iterator
}

type IterHandler struct {
	release capnp.ReleaseFunc
}

func (ih *IterHandler) Handle(ctx context.Context, call api.Routing_Handler_handle) error{
	// TODO
	return nil
}

func (ih *IterHandler) Next(){

}

func (ih *IterHandler) Record() Record{
	
}

func (ih *IterHandler) Deadline() time.Time{
	
}

func (ih *IterHandler) Finish(){
	if ih.release !=nil{
		ih.release()
		ih.release = nil
	}
}

type Iterator struct {
	it api.Iterator
}

func (it Iterator) Next(ctx context.Context, amount int64) {
	_, release := it.it.Next(ctx, func(i api.Iterator_next_Params) error {
		i.SetAmount(amount)
		return nil
	})
	defer release()
}

func (it Iterator) Records(ctx context.Context) ([]cluster.Record, error) {
	fut, _ := it.it.Records(ctx, func(i api.Iterator_records_Params) error {
		return nil
	})
	// defer release()

	recs, err := FutureRecords{fut}.Records()
	println("A", recs[0].Peer())
	return recs, err
}

func (it Iterator) Deadline(ctx context.Context) time.Time {
	fut, release := it.it.Deadline(ctx, func(i api.Iterator_deadline_Params) error {
		return nil
	})
	defer release()

	results, err := fut.Struct()
	if err != nil {
		return time.Time{}
	}

	return time.UnixMicro(results.Deadline())
}

func (it Iterator) Finish(ctx context.Context) {
	_, release := it.it.Finish(ctx, func(i api.Iterator_finish_Params) error {
		return nil
	})
	defer release()
}

type FutureRecords struct {
	recs api.Iterator_records_Results_Future
}

func (fr FutureRecords) Records() ([]cluster.Record, error) {
	results, err := fr.recs.Struct()
	if err != nil {
		return nil, err
	}
	reclist, err := results.Records()
	if err != nil {
		return nil, err
	}

	recs := make([]cluster.Record, 0, reclist.Len())
	for i := 0; i < reclist.Len(); i++ {
		println(reclist.At(i).Peer())
		recs = append(recs, Record{reclist.At(i)})
	}
	println(recs[0])
	println(recs[0].Peer())
	return recs, nil
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
	return Record{rec}, s.Ok(), nil
}

type Record struct {
	rec api.Record
}

func (rec Record) Peer() peer.ID {
	peerID, err := rec.rec.Peer()
	if err != nil {
		return ""
	}
	return peer.ID(peerID)
}

func (rec Record) TTL() time.Duration {
	return time.Duration(rec.rec.Ttl())
}

func (rec Record) Seq() uint64 {
	return rec.Seq()
}
