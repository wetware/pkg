package host

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	capnp "zombiezen.com/go/capnproto2"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/internal/filter"
	"github.com/wetware/ww/pkg/internal/rpc"
	"github.com/wetware/ww/pkg/internal/rpc/anchor"
	"github.com/wetware/ww/pkg/internal/tree"
	"github.com/wetware/ww/pkg/lang"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

/*
	api.go contains the capnp api that is served by the host
*/

var (
	_ ww.Anchor = (*rootAnchor)(nil)
	_ ww.Anchor = (*hostAnchor)(nil)

	_ rpc.Capability = (*rootAnchorCap)(nil)

	_ api.Anchor_Server = (*rootAnchorCap)(nil)
	_ api.Anchor_Server = (*anchorCap)(nil)
)

type anchorParams struct {
	fx.In

	Host   host.Host
	Filter filter.Filter
}

type anchorOut struct {
	fx.Out

	Tree    tree.Node
	Handler rpc.Capability `group:"rpc"`
	Root    ww.Anchor
}

func newAnchor(ps anchorParams) (out anchorOut) {
	out.Tree = tree.New()
	out.Handler = rootAnchorCap{
		id:           ps.Host.ID(),
		routingTable: ps.Filter,
		anchorCap:    anchorCap{root: out.Tree},
	}
	out.Root = rootAnchor{
		local: ps.Host.ID().String(),
		node:  out.Tree,
		term:  rpc.NewTerminal(ps.Host),
	}
	return
}

type routingTable interface {
	Peers() peer.IDSlice
}

type rootAnchor struct {
	local string
	node  tree.Node
	term  rpc.Terminal
}

func (rootAnchor) String() string {
	return "/"
}

func (root rootAnchor) Path() []string {
	return []string{}
}

func (root rootAnchor) Ls(ctx context.Context) ([]ww.Anchor, error) {
	return anchor.Ls(ctx, root.term, rpc.AutoDial{})
}

func (root rootAnchor) Walk(ctx context.Context, path []string) ww.Anchor {
	if anchorpath.Root(path) {
		return root
	}

	if root.isLocal(path) {
		return hostAnchor{
			root: path[0],
			node: root.node.Walk(path[1:]),
		}
	}

	return anchor.Walk(ctx, root.term, rpc.DialString(path[0]), path)
}

func (root rootAnchor) Load(context.Context) (ww.Any, error) {
	// TODO:  return a dict with some info about the global cluster
	return nil, errors.New("NOT IMPLEMENTED")
}

func (root rootAnchor) Store(context.Context, ww.Any) error {
	return errors.New("not implemented")
}

func (root rootAnchor) Go(context.Context, ww.ProcSpec) error {
	return errors.New("not implemented")
}

func (root rootAnchor) isLocal(path []string) bool {
	return !anchorpath.Root(path) && path[0] == root.local
}

type hostAnchor struct {
	root string
	node tree.Node
}

func (a hostAnchor) String() string {
	return anchorpath.Join(a.Path())
}

func (a hostAnchor) Path() []string {
	return append([]string{a.root}, a.node.Path()...)
}

func (a hostAnchor) Ls(context.Context) ([]ww.Anchor, error) {
	ns := a.node.List()
	as := make([]ww.Anchor, len(ns))
	for i, n := range ns {
		as[i] = hostAnchor{root: a.root, node: n}
	}

	return as, nil
}

func (a hostAnchor) Walk(_ context.Context, path []string) ww.Anchor {
	return hostAnchor{
		root: a.root,
		node: a.node.Walk(path),
	}
}

func (a hostAnchor) Load(context.Context) (ww.Any, error) {
	if n := a.node.Load(); n != nil {
		return lang.LiftValue(*n)
	}

	return lang.Nil{}, nil
}

func (a hostAnchor) Store(_ context.Context, any ww.Any) error {
	if v := any.Value(); a.node.Store(&v) {
		return nil
	}

	return ww.ErrAnchorNotEmpty
}

func (a hostAnchor) Go(_ context.Context, s ww.ProcSpec) error {
	panic("NOT IMPLEMENTED")
}

type rootAnchorCap struct {
	id peer.ID
	routingTable
	anchorCap
}

func (r rootAnchorCap) Loggable() map[string]interface{} {
	return map[string]interface{}{"cap": "root_anchor"}
}

func (r rootAnchorCap) Protocol() protocol.ID {
	return ww.AnchorProtocol
}

func (r rootAnchorCap) Client() capnp.Client {
	return api.Anchor_ServerToClient(r).Client
}

func (r rootAnchorCap) Ls(call api.Anchor_ls) error {
	peers := r.Peers()

	cs, err := call.Results.NewChildren(int32(len(peers)))
	if err != nil {
		return errinternal(err)
	}

	for i, p := range peers {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return errinternal(err)
		}

		a, err := api.NewAnchor_SubAnchor(seg)
		if err != nil {
			return errinternal(err)
		}

		a.SetRoot()

		if err = a.SetPath(p.String()); err != nil {
			return errinternal(err)
		}

		if err = cs.Set(i, a); err != nil {
			return errinternal(err)
		}
	}

	return nil
}

func (r rootAnchorCap) Walk(call api.Anchor_walk) error {
	path, err := call.Params.Path()
	if err != nil {
		return errinternal(err)
	}

	parts := anchorpath.Parts(path)
	if id := parts[0]; id != r.id.String() {
		return errors.Errorf("bad request: id mismatch (expected %s, got %s)", r.id, id)
	}

	// pop the path head before passing the call down to the `anchorCap`.
	if err = call.Params.SetPath(anchorpath.Join(parts[1:])); err != nil {
		return errinternal(err)
	}

	return r.anchorCap.Walk(call)
}

func (r rootAnchorCap) Load(call api.Anchor_load) error {
	return errors.New("NOT IMPLEMENTED")
}

func (r rootAnchorCap) Store(call api.Anchor_store) error {
	return errors.New("NOT IMPLEMENTED")
}

type anchorCap struct {
	root tree.Node
}

func (a anchorCap) Loggable() map[string]interface{} {
	return map[string]interface{}{"cap": "anchorCap"}
}

func (a anchorCap) Client() capnp.Client {
	return api.Anchor_ServerToClient(a).Client
}

func (a anchorCap) Ls(call api.Anchor_ls) error {
	children := a.root.List()

	cs, err := call.Results.NewChildren(int32(len(children)))
	if err != nil {
		return errinternal(err)
	}

	for i, child := range a.root.List() {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			break
		}

		item, err := api.NewRootAnchor_SubAnchor(seg)
		if err != nil {
			break
		}

		if err = item.SetPath(child.Name); err != nil {
			break
		}

		if err = item.SetAnchor(a.subAnchor(child)); err != nil {
			break
		}

		if err = cs.Set(i, item); err != nil {
			break
		}
	}

	return errinternal(err)
}

func (a anchorCap) Walk(call api.Anchor_walk) error {
	path, err := call.Params.Path()
	if err != nil {
		return errinternal(err)
	}

	child := a.subAnchor(a.root.Walk(anchorpath.Parts(path)))
	return errinternal(call.Results.SetAnchor(child))
}

func (a anchorCap) Load(call api.Anchor_load) error {
	if v := a.root.Load(); v != nil {
		return call.Results.SetValue(*v)
	}

	return call.Results.SetValue(lang.Nil{}.Value())
}

func (a anchorCap) Store(call api.Anchor_store) error {
	v, err := call.Params.Value()
	if err != nil {
		return err
	}

	if v.Which() == api.Value_Which_nil {
		_ = a.root.Store(nil) // scrub never fails
	}

	if a.root.Store(&v) {
		return nil
	}

	return ww.ErrAnchorNotEmpty
}

func (a anchorCap) subAnchor(node tree.Node) api.Anchor {
	return api.Anchor_ServerToClient(anchorCap{
		root: node,
	})
}

func (a anchorCap) Go(call api.Anchor_go) error {
	spec, err := call.Params.Spec()
	if err != nil {
		return err
	}

	switch spec.Which() {
	case api.Anchor_ProcSpec_Which_goroutine:
		return errors.New("Goroutine NOT IMPLEMENTED")

	case api.Anchor_ProcSpec_Which_osProc:
		return errors.New("UnixProc NOT IMPLEMENTED")

	case api.Anchor_ProcSpec_Which_docker:
		return errors.New("Docker NOT IMPLEMENTED")

	default:
		return errors.New("invalid spec")
	}
}

func errinternal(err error) error {
	return errors.Wrap(err, "internal server error")
}
