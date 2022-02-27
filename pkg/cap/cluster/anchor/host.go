package anchor

import (
	"context"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	api "github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/vat"
)

var (
	Capability = vat.BasicCap{
		"hostAnchor/packed",
		"hostAnchor"}
	defaultPolicy = server.Policy{
		// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
		//        https://github.com/capnproto/go-capnproto2/issues/189
		MaxConcurrentCalls: 64,
		AnswerQueueSize:    64,
	}
)

type HostAnchor struct {
	Peer peer.ID
	Vat  vat.Network

	client api.Host

	once sync.Once
}

func (ha HostAnchor) Ls(ctx context.Context, path []string) (AnchorIterator, error) {
	if err := ha.bootstrapOnce(ctx); err != nil {
		return nil, err
	}

	anchor := api.Anchor{Client: ha.client.Client}
	return newIterator(ctx, anchor, path)
}

func (ha HostAnchor) Walk(ctx context.Context, path []string) (Anchor, error) {
	if err := ha.bootstrapOnce(ctx); err != nil {
		return nil, err
	}

	fut, release := ha.client.Walk(ctx, func(a api.Anchor_walk_Params) error {
		capPath, err := a.NewPath(int32(len(path)))
		if err != nil {
			return err
		}
		for i, e := range path {
			if err := capPath.Set(i, e); err != nil {
				return err
			}
		}
		return nil
	})

	return ContainerAnchor{fut: fut, release: release}, nil
}

func (ha HostAnchor) bootstrapOnce(ctx context.Context) error {
	var (
		conn *rpc.Conn
		err  error
	)

	ha.once.Do(func() {
		conn, err = ha.Vat.Connect(
			ctx,
			peer.AddrInfo{ID: ha.Peer},
			Capability,
		)
		if err != nil {
			return
		}
		ha.client = api.Host{Client: conn.Bootstrap(ctx)}
	})

	return err
}

type HostAnchorIterator struct {
	Vat     vat.Network
	It      *cluster.Iterator
	Release capnp.ReleaseFunc
}

func (hai HostAnchorIterator) Next(ctx context.Context) error {
	hai.It.Next(ctx)
	return hai.It.Err
}

func (hai HostAnchorIterator) Finish() {
	hai.Release()
}

func (hai HostAnchorIterator) Anchor() Anchor {
	return HostAnchor{
		Peer: hai.It.Record().Peer(),
		Vat:  hai.Vat,
	}
}

type HostAnchorServer struct {
	vat vat.Network

	tree *Node
}

func newHostAnchorServer(vat vat.Network, tree *Node) (HostAnchorServer, error) {
	sv := HostAnchorServer{vat: vat, tree: tree}
	vat.Export(Capability, sv)
	return sv, nil
}

func (sv HostAnchorServer) Host(ctx context.Context, call api.Host_host) error {
	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	return results.SetHost(sv.vat.Host.ID().String())
}

func (sv HostAnchorServer) Ls(ctx context.Context, call api.Anchor_ls) error {
	b := newBatcher(call.Args().Handler())

	for it := sv.tree.Ls(ctx); it.Node() != nil; it.Next() {
		if err := b.Send(ctx, it.Node().Anchor(), it.Node().Name()); err != nil {
			it.Finish()
			return err
		}
	}

	return b.Wait(ctx)
}

func (sv HostAnchorServer) Walk(ctx context.Context, call api.Anchor_walk) error {
	capPath, err := call.Args().Path()
	if err != nil {
		return err
	}

	path := make([]string, 0, capPath.Len())
	for i := 0; i < capPath.Len(); i++ {
		e, err := capPath.At(i)
		if err != nil {
			return err
		}
		path = append(path, e)
	}
	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	return results.SetAnchor(sv.tree.Walk(ctx, path).Anchor())
}

func (sv HostAnchorServer) Client() *capnp.Client {
	return api.Host_ServerToClient(sv, &defaultPolicy).Client
}
