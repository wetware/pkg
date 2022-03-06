package cluster

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	api "github.com/wetware/ww/internal/api/cluster"
)

var (
	containerDefaultPolicy = server.Policy{
		// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
		//        https://github.com/capnproto/go-capnproto2/issues/189
		MaxConcurrentCalls: 64,
		AnswerQueueSize:    64,
	}
)

type containerAnchor struct {
	path   []string
	client api.Container

	release capnp.ReleaseFunc
}

func (ca containerAnchor) Name() string {
	n := len(ca.Path()) - 1
	if n >= 0 {
		return ca.path[n]
	}
	return ""
}

func (ca containerAnchor) Path() []string {
	return ca.path
}

func (ca containerAnchor) Ls(ctx context.Context) (AnchorIterator, error) {
	fut, release := ca.client.Ls(ctx, func(a api.Anchor_ls_Params) error {
		return nil
	})
	select {
	case <-fut.Done():
		results, err := fut.Struct()
		if err != nil {
			return nil, err
		}
		children, err := results.Children()
		if err != nil {
			return nil, err
		} else {
			return &containerAnchorIterator{path: ca.Path(), children: children, release: release}, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (ca containerAnchor) Walk(ctx context.Context, path []string) (Anchor, error) {
	if len(path) == 0 {
		return ca, nil
	}

	fut, release := ca.client.Walk(ctx, func(a api.Anchor_walk_Params) error {
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
	return containerAnchor{path: append(ca.Path(), path...), client: api.Container(fut.Anchor()), release: release}, nil
}

func (ca containerAnchor) Set(ctx context.Context, data []byte) error {
	c := api.Container{Client: ca.client.Client}
	fut, release := c.Set(ctx, func(c api.Container_set_Params) error {
		return c.SetData(data)
	})
	defer release()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-fut.Done():
		_, err := fut.Struct()
		return err
	}
}

func (ca containerAnchor) Get(ctx context.Context) (data []byte, release func()) {
	c := api.Container{Client: ca.client.Client}
	fut, release := c.Get(ctx, func(c api.Container_get_Params) error {
		return nil
	})
	select {
	case <-ctx.Done():
		return nil, release
	case <-fut.Done():
		results, err := fut.Struct()
		if err != nil {
			return nil, release
		}
		data, _ := results.Data()
		return data, release

	}
}

type containerAnchorIterator struct {
	path []string

	i        int
	children api.Anchor_Child_List
	release  capnp.ReleaseFunc

	err error
}

func (it *containerAnchorIterator) Next(context.Context) bool {
	it.i++
	return it.i <= it.children.Len()
}

func (it *containerAnchorIterator) Finish() {
	// TODO
}
func (it *containerAnchorIterator) Anchor() Anchor {
	child := it.children.At(it.i - 1)
	name, err := child.Name()
	if err != nil {
		it.err = err
		return nil
	}

	return containerAnchor{path: append(it.path, name), client: api.Container(child.Anchor())}
}
func (it *containerAnchorIterator) Err() error {
	return it.err
}

type containerAnchorServer struct {
	tree *node

	client api.Container
}

func newContainerServer(n *node) *containerAnchorServer {
	sv := containerAnchorServer{tree: n}
	sv.client = api.Container_ServerToClient(&sv, &defaultPolicy)
	return &sv
}

func (sv *containerAnchorServer) Ls(ctx context.Context, call api.Anchor_ls) error {
	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	children, release := sv.tree.Children()
	defer release()

	capChildren, err := results.NewChildren(int32(len(children)))
	if err != nil {
		return err
	}

	i := 0
	for name, child := range children {
		capChild := capChildren.At(i)
		if err := capChild.SetAnchor(child.Server.Anchor()); err != nil {
			return err
		}

		if err := capChild.SetName(name); err != nil {
			return err
		}
		i++
	}

	return nil
}

func (sv *containerAnchorServer) Walk(ctx context.Context, call api.Anchor_walk) error {
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
	node := sv.tree.Walk(path)
	if node.Server == nil {
		node.Server = newContainerServer(node)
	}
	return results.SetAnchor(node.Server.Anchor())
}

func (sv *containerAnchorServer) Get(ctx context.Context, call api.Container_get) error {
	data, ok := sv.tree.Value.Load().([]byte)
	if !ok {
		return nil
	}
	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	return results.SetData(data)
}

func (sv *containerAnchorServer) Set(ctx context.Context, call api.Container_set) error {
	data, err := call.Args().Data()
	if err != nil {
		return err
	}
	sv.tree.Value.Store(data)
	return nil
}

func (sv *containerAnchorServer) Anchor() api.Anchor {
	return api.Anchor(sv.client)
}

func (sv *containerAnchorServer) Shutdown() {
	// TODO
}
