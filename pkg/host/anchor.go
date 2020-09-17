package host

import (
	"github.com/pkg/errors"

	capnp "zombiezen.com/go/capnproto2"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

/*
	api.go contains the capnp api that is served by the host
*/

var (
	_ api.Anchor_Server = (*rootAnchor)(nil)
	_ api.Anchor_Server = (*anchor)(nil)

	nullValue api.Value
)

func init() {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	if nullValue, err = api.NewRootValue(seg); err != nil {
		panic(err)
	}

	nullValue.SetNil()
}

type routingTable interface {
	Peers() peer.IDSlice
}

type rootAnchor struct {
	id peer.ID
	routingTable
	anchor
}

func newRootAnchor(host host.Host, t routingTable) rootAnchor {
	return rootAnchor{
		id:           host.ID(),
		routingTable: t,
		anchor: anchor{
			root: newAnchorTree(),
		},
	}
}

func (r rootAnchor) Loggable() map[string]interface{} {
	return map[string]interface{}{"cap": "root_anchor"}
}

func (r rootAnchor) Protocol() protocol.ID {
	return ww.Protocol
}

func (r rootAnchor) Client() capnp.Client {
	return api.Anchor_ServerToClient(r).Client
}

func (r rootAnchor) Ls(call api.Anchor_ls) error {
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

func (r rootAnchor) Walk(call api.Anchor_walk) error {
	path, err := call.Params.Path()
	if err != nil {
		return errinternal(err)
	}

	parts := anchorpath.Parts(path)
	if id := parts[0]; id != r.id.String() {
		return errors.Errorf("bad request: id mismatch (expected %s, got %s)", r.id, id)
	}

	// pop the path head before passing the call down to the `anchor`.
	if err = call.Params.SetPath(anchorpath.Join(parts[1:])); err != nil {
		return errinternal(err)
	}

	return r.anchor.Walk(call)
}

func (r rootAnchor) Load(call api.Anchor_load) error {
	return errors.New("NOT IMPLEMENTED")
}

func (r rootAnchor) Store(call api.Anchor_store) error {
	return errors.New("NOT IMPLEMENTED")
}

type anchor struct {
	root anchorNode
}

func (a anchor) Loggable() map[string]interface{} {
	return map[string]interface{}{"cap": "anchor"}
}

func (a anchor) Client() capnp.Client {
	return api.Anchor_ServerToClient(a).Client
}

func (a anchor) Ls(call api.Anchor_ls) error {
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

		if err = item.SetPath(child.Path); err != nil {
			break
		}

		if err = item.SetAnchor(a.subAnchor(child.Node)); err != nil {
			break
		}

		if err = cs.Set(i, item); err != nil {
			break
		}
	}

	return errinternal(err)
}

func (a anchor) Walk(call api.Anchor_walk) error {
	path, err := call.Params.Path()
	if err != nil {
		return errinternal(err)
	}

	child := a.subAnchor(a.root.Walk(anchorpath.Parts(path)))
	return errinternal(call.Results.SetAnchor(child))
}

func (a anchor) Load(call api.Anchor_load) error {
	if v := a.root.Load(); v != nil {
		return call.Results.SetValue(*v)
	}

	return call.Results.SetValue(nullValue)
}

func (a anchor) Store(call api.Anchor_store) error {
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

	return errors.New("anchor contains value")
}

func (a anchor) subAnchor(node anchorNode) api.Anchor {
	return api.Anchor_ServerToClient(anchor{
		root: node,
	})
}

func errinternal(err error) error {
	return errors.Wrap(err, "internal server error")
}
