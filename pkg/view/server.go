package view

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"

	api "github.com/wetware/ww/api/cluster"
	"github.com/wetware/ww/cluster/pulse"
	"github.com/wetware/ww/cluster/query"
	"github.com/wetware/ww/cluster/routing"
)

type RecordBinder interface {
	BindRecord(api.View_Record) error
}

type Server struct {
	RoutingTable interface {
		Snapshot() routing.Snapshot
	}
}

func (s Server) Client() capnp.Client {
	return capnp.Client(s.View())
}

func (s Server) View() View {
	return View(api.View_ServerToClient(s))
}

func (s Server) Lookup(ctx context.Context, call api.View_lookup) error {
	sel, err := call.Args().Selector()
	if err == nil {
		err = s.bind(maybeRecord(call), selector(sel).Bind(query.First()))
	}

	return err
}

func (s Server) Iter(ctx context.Context, call api.View_iter) error {
	sel, err := call.Args().Selector()
	if err != nil {
		return err
	}

	var (
		handler = call.Args().Handler()
		// TODO(soon): use BBR once scheduler bug is fixed

		iter = iterator(ctx, handler)
	)

	if err = s.bind(iter, selector(sel)); err == nil {
		call.Go()
		err = handler.WaitStreaming()
	}

	return err
}

func (s Server) Reverse(ctx context.Context, call api.View_reverse) error {
	return fmt.Errorf("NOT IMPLEMENTED") // TODO(soon):  implement Reverse()
}

func selector(s api.View_Selector) query.Selector {
	switch s.Which() {
	case api.View_Selector_Which_all:
		return query.All()

	case api.View_Selector_Which_match:
		match, err := s.Match()
		if err != nil {
			return query.Failure(err)
		}

		return query.Select(index{match})

	case api.View_Selector_Which_from:
		from, err := s.From()
		if err != nil {
			return query.Failure(err)
		}

		return query.From(index{from})
	}

	return query.Failuref("invalid selector: %s", s.Which())
}

// binds a record
type bindFunc func(routing.Record) error

func (s Server) bind(bind bindFunc, selector query.Selector) error {
	it, err := selector(s.RoutingTable.Snapshot())
	if err != nil {
		return err
	}

	for r := it.Next(); r != nil; r = it.Next() {
		if err = bind(r); err != nil {
			break
		}
	}

	return err
}

func maybeRecord(call api.View_lookup) bindFunc {
	return func(r routing.Record) error {
		res, err := call.AllocResults()
		if err != nil {
			return err
		}

		return maybe(res, r)
	}
}

func iterator(ctx context.Context, h api.View_Handler) bindFunc {
	return func(r routing.Record) error {
		h.Recv(ctx, record(r))
		return nil
	}
}

func record(r routing.Record) func(api.View_Handler_recv_Params) error {
	return func(ps api.View_Handler_recv_Params) error {
		rec, err := ps.NewRecord()
		if err != nil {
			return err
		}

		return copyRecord(rec, r)
	}
}

func maybe(res api.View_lookup_Results, r routing.Record) error {
	if r == nil {
		return nil
	}

	result, err := res.NewResult()
	if err != nil {
		return err
	}

	rec, err := result.NewJust()
	if err != nil {
		return err
	}

	return copyRecord(rec, r)
}

func copyRecord(rec api.View_Record, r routing.Record) error {
	if b, ok := r.(RecordBinder); ok {
		return b.BindRecord(rec)
	}

	err := rec.SetPeer(string(r.Peer()))
	if err != nil {
		return err
	}

	var hb pulse.Heartbeat
	if hb.Heartbeat, err = rec.NewHeartbeat(); err != nil {
		return err
	}

	rec.SetSeq(r.Seq())
	hb.SetTTL(r.TTL())
	hb.SetServer(r.Server())

	if err := copyHost(hb, r); err != nil {
		return err
	}

	return copyMeta(hb, r)
}

func copyHost(rec pulse.Heartbeat, r routing.Record) error {
	name, err := r.Host()
	if err == nil {
		err = rec.SetHost(name)
	}

	return err
}

func copyMeta(rec pulse.Heartbeat, r routing.Record) error {
	meta, err := r.Meta()
	if err == nil {
		err = rec.Heartbeat.SetMeta(capnp.TextList(meta))
	}

	return err
}

type index struct{ api.View_Index }

func (ix index) String() string {
	if ix.Which() == api.View_Index_Which_peer {
		return "id"
	}

	return ix.Which().String()
}

func (ix index) ServerBytes() ([]byte, error) {
	return ix.View_Index.Server()
}
