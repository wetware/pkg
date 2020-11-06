package host

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spy16/slurp/core"
	"go.uber.org/fx"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/server"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/cluster"
	"github.com/wetware/ww/pkg/internal/rpc"
	"github.com/wetware/ww/pkg/internal/rpc/anchor"
	"github.com/wetware/ww/pkg/internal/tree"
	"github.com/wetware/ww/pkg/lang/builtin"
	"github.com/wetware/ww/pkg/mem"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

/*
	api.go contains the capnp api that is served by the host
*/

var (
	_ ww.Anchor = (*rootAnchor)(nil)
	_ ww.Anchor = (*localAnchor)(nil)

	_ rpc.Capability = (*rootAnchorCap)(nil)

	_ api.Anchor_Server = (*rootAnchorCap)(nil)
	_ api.Anchor_Server = (*anchorCap)(nil)
)

type anchorParams struct {
	fx.In

	Log     ww.Logger
	Host    host.Host
	Cluster cluster.PeerSet
}

type anchorOut struct {
	fx.Out

	Handler rpc.Capability `group:"rpc"`
}

func newAnchor(ps anchorParams) (out anchorOut) {
	out.Handler = rootAnchorCap{
		root: newRootAnchor(ps.Log, ps.Cluster, ps.Host),
	}

	return
}

type rootAnchor struct {
	log ww.Logger
	// env *parens.Env
	peerProvider

	localPath string
	node      tree.Node
	term      rpc.Terminal
}

func newRootAnchor(log ww.Logger, ps peerProvider, h host.Host) *rootAnchor {
	root := &rootAnchor{
		log:          log.WithField("path", "/"),
		peerProvider: ps,
		localPath:    h.ID().String(),
		node:         tree.New(),
		term:         rpc.NewTerminal(h),
	}

	// root.env = lang.New(root)
	return root
}

func (rootAnchor) Name() string        { return "" }
func (root rootAnchor) Path() []string { return []string{} } // TODO: return nil

func (root rootAnchor) Ls(ctx context.Context) ([]ww.Anchor, error) {
	peers := root.Peers()

	as := make([]ww.Anchor, len(peers))
	for i, p := range peers {
		as[i] = anchor.NewHost(root.term, p)
	}

	return as, nil
}

func (root rootAnchor) Walk(ctx context.Context, path []string) ww.Anchor {
	if anchorpath.Root(path) {
		return root
	}

	if root.isLocal(path) {
		return localAnchor{
			log: root.log.WithField("path", anchorpath.Join(path)),
			// env:  root.env,
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

func (root rootAnchor) Go(context.Context, ...ww.Any) (ww.Any, error) {
	return nil, errors.New("not implemented")
}

func (root rootAnchor) isLocal(path []string) bool {
	return !anchorpath.Root(path) && path[0] == root.localPath
}

type localAnchor struct {
	log  ww.Logger
	root string
	node tree.Node
	env  core.Env
}

func (a localAnchor) String() string {
	return anchorpath.Join(a.Path())
}

func (a localAnchor) Path() []string {
	return append([]string{a.root}, a.node.Path()...)
}

func (a localAnchor) Name() string { return a.node.Name }

func (a localAnchor) Ls(context.Context) ([]ww.Anchor, error) {
	ns := a.node.List()
	as := make([]ww.Anchor, len(ns))
	for i, n := range ns {
		as[i] = localAnchor{root: a.root, node: n}
	}

	return as, nil
}

func (a localAnchor) Walk(_ context.Context, path []string) ww.Anchor {
	return localAnchor{
		root: a.root,
		node: a.node.Walk(path),
	}
}

func (a localAnchor) Load(context.Context) (ww.Any, error) {
	if n := a.node.Load(); !n.Nil() {
		return builtin.AsAny(n)
	}

	return builtin.Nil{}, nil
}

func (a localAnchor) Store(_ context.Context, any ww.Any) error {
	if v := any.MemVal(); a.node.Store(v) {
		return nil
	}

	return ww.ErrAnchorNotEmpty
}

func (a localAnchor) Go(_ context.Context, args ...ww.Any) (p ww.Any, err error) {
	return nil, errors.New("Host Interpreter NOT IMPLEMENTED (pkg/host/anchor.go")
	// a.node.Txn(func(t tree.Transaction) {
	// 	if err = ww.ErrAnchorNotEmpty; t.Load().Nil() {
	// 		if p, err = proc.Spawn(a.env.Fork(), args...); err == nil {
	// 			_ = t.Store(p.MemVal())
	// 			a.log.WithField("args", args).Info("process started")
	// 		}
	// 	}
	// })
	//
	// return
}

type rootAnchorCap struct{ root *rootAnchor }

func (rootAnchorCap) Loggable() map[string]interface{} {
	return map[string]interface{}{"cap": "anchor"}
}

func (rootAnchorCap) Protocol() protocol.ID {
	return ww.AnchorProtocol
}

func (a rootAnchorCap) Client() *capnp.Client {
	return api.Anchor_ServerToClient(a, &server.Policy{}).Client
}

func (a rootAnchorCap) Ls(ctx context.Context, call api.Anchor_ls) error {
	hosts, err := a.root.Ls(ctx)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	cs, err := res.NewChildren(int32(len(hosts)))
	if err != nil {
		return errmem(err)
	}

	for i, h := range hosts {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return errmem(err)
		}

		a, err := api.NewAnchor_SubAnchor(seg)
		if err != nil {
			return errmem(err)
		}

		a.SetRoot()

		if err = a.SetPath(anchorpath.Join(h.Path())); err != nil {
			return errmem(err)
		}

		if err = cs.Set(i, a); err != nil {
			return errmem(err)
		}
	}

	return nil
}

func (a rootAnchorCap) Walk(ctx context.Context, call api.Anchor_walk) error {
	path, err := call.Args().Path()
	if err != nil {
		return errmem(err)
	}

	parts := anchorpath.Parts(path)

	// belt-and-suspenders
	if !a.root.isLocal(parts) {
		return errors.Errorf("misrouted RPC: host %s received RPC at path %s",
			a.root.localPath,
			parts[0])
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	err = res.SetAnchor(api.Anchor_ServerToClient(
		anchorCap{a.root.Walk(ctx, parts)},
		&server.Policy{},
	))

	return errmem(err)
}

func (a rootAnchorCap) Load(ctx context.Context, call api.Anchor_load) error {
	any, err := a.root.Load(ctx)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetValue(any.MemVal().Raw)
}

func (a rootAnchorCap) Store(ctx context.Context, call api.Anchor_store) error {
	return a.root.Store(ctx, nil)
}

func (a rootAnchorCap) Go(ctx context.Context, call api.Anchor_go) error {
	vs, err := call.Args().Args()
	if err != nil {
		return err
	}

	args := make([]ww.Any, vs.Len())
	for i := 0; i < vs.Len(); i++ {
		if args[i], err = builtin.AsAny(mem.Value{Raw: vs.At(i)}); err != nil {
			return err
		}
	}

	p, err := a.root.Go(ctx, args...)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetProc(p.MemVal().Raw.Proc())
}

type anchorCap struct{ anchor ww.Anchor }

func (a anchorCap) Ls(ctx context.Context, call api.Anchor_ls) error {
	as, err := a.anchor.Ls(ctx)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	cs, err := res.NewChildren(int32(len(as)))
	if err != nil {
		return errmem(err)
	}

	for i, child := range as {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			break
		}

		item, err := api.NewRootAnchor_SubAnchor(seg)
		if err != nil {
			break
		}

		if err = item.SetPath(child.Name()); err != nil {
			break
		}

		if err = item.SetAnchor(api.Anchor_ServerToClient(anchorCap{child}, &server.Policy{})); err != nil {
			break
		}

		if err = cs.Set(i, item); err != nil {
			break
		}
	}

	return errmem(err)
}

func (a anchorCap) Walk(ctx context.Context, call api.Anchor_walk) error {
	path, err := call.Args().Path()
	if err != nil {
		return errmem(err)
	}

	sub := a.anchor.Walk(nil, anchorpath.Parts(path))

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	err = res.SetAnchor(api.Anchor_ServerToClient(anchorCap{sub}, &server.Policy{}))
	return errmem(err)
}

func (a anchorCap) Load(ctx context.Context, call api.Anchor_load) error {
	any, err := a.anchor.Load(ctx)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetValue(any.MemVal().Raw)
}

func (a anchorCap) Store(ctx context.Context, call api.Anchor_store) error {
	raw, err := call.Args().Value()
	if err == nil {
		return err
	}

	any, err := builtin.AsAny(mem.Value{Raw: raw})
	if err != nil {
		return err
	}

	return a.anchor.Store(ctx, any)
}

func (a anchorCap) Go(ctx context.Context, call api.Anchor_go) error {
	vs, err := call.Args().Args()
	if err != nil {
		return err
	}

	args := make([]ww.Any, vs.Len())
	for i := 0; i < vs.Len(); i++ {
		if args[i], err = builtin.AsAny(mem.Value{Raw: vs.At(i)}); err != nil {
			return err
		}
	}

	p, err := a.anchor.Go(ctx, args...)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetProc(p.MemVal().Raw.Proc())
}

func errmem(err error) error {
	return errors.Wrap(err, "remote memory error")
}
