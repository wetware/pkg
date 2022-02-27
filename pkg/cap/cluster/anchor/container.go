package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	api "github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/vat"
)

var (
	ContainerCapability = vat.BasicCap{
		"containerAnchor/packed",
		"containerAnchor"}
	containerDefaultPolicy = server.Policy{
		// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
		//        https://github.com/capnproto/go-capnproto2/issues/189
		MaxConcurrentCalls: 64,
		AnswerQueueSize:    64,
	}
)

const (
	defaultBatchSize   = 64
	defaultMaxInflight = 8
)

type ContainerAnchor struct {
	path   []string
	client api.Container

	fut     api.Anchor_walk_Results_Future
	release capnp.ReleaseFunc
}

func (ca ContainerAnchor) Path() []string {
	return ca.path
}

func (ca ContainerAnchor) Ls(ctx context.Context) (AnchorIterator, error) {
	return newIterator(ctx, ca.fut.Anchor(), ca.Path())
}

func (ca ContainerAnchor) Walk(ctx context.Context, path []string) (Anchor, error) {
	fut, release := ca.fut.Anchor().Walk(ctx, func(a api.Anchor_walk_Params) error {
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
	return ContainerAnchor{path: append(ca.Path(), path...), fut: fut, release: release}, nil
}

func (ca ContainerAnchor) Set(ctx context.Context, data []byte) error {
	c := api.Container{Client: ca.fut.Anchor().Client}
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

func (ca ContainerAnchor) Get(ctx context.Context) (data []byte, release func()) {
	c := api.Container{Client: ca.fut.Anchor().Client}
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
	return nil, nil // TODO
}

type ContainerAnchorServer struct {
	tree *node

	client api.Container
}

func newContainerServer(n *node) *ContainerAnchorServer {
	sv := ContainerAnchorServer{tree: n}
	sv.client = api.Container_ServerToClient(&sv, &defaultPolicy)
	return &sv
}

func (sv *ContainerAnchorServer) Ls(ctx context.Context, call api.Anchor_ls) error {
	b := newBatcher(call.Args().Handler())

	children, release := sv.tree.Children()
	defer release()

	for name, child := range children {
		if err := b.Send(ctx, child.Server.Client(), name); err != nil {
			return err
		}
	}

	return b.Wait(ctx)
}

func (sv *ContainerAnchorServer) Walk(ctx context.Context, call api.Anchor_walk) error {
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
	return results.SetAnchor(node.Server.Client())
}

func (sv *ContainerAnchorServer) Get(ctx context.Context, call api.Container_get) error {
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

func (sv *ContainerAnchorServer) Set(ctx context.Context, call api.Container_set) error {
	data, err := call.Args().Data()
	if err != nil {
		return err
	}
	sv.tree.Value.Store(data)
	return nil
}

func (sv *ContainerAnchorServer) Client() api.Anchor {
	return api.Anchor(sv.client)
}

func (sv *ContainerAnchorServer) Shutdown() {
	// TODO
}
