package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	api "github.com/wetware/ww/internal/api/cluster"
)

const (
	defaultBatchSize   = 64
	defaultMaxInflight = 8
)

type ContainerAnchor struct {
	client api.Container

	fut     api.Anchor_walk_Results_Future
	release capnp.ReleaseFunc
}

func (ca ContainerAnchor) Ls(ctx context.Context, path []string) (AnchorIterator, error) {
	h := make(handler, defaultMaxInflight)

	fut, release := ca.fut.Anchor().Ls(ctx, func(a api.Anchor_ls_Params) error {
		capPath, err := a.NewPath(int32(len(path)))
		if err != nil {
			return err
		}
		for i, e := range path {
			if err := capPath.Set(i, e); err != nil {
				return err
			}
		}

		c := api.Anchor_Handler_ServerToClient(h, &server.Policy{
			MaxConcurrentCalls: cap(h),
			AnswerQueueSize:    cap(h),
		})
		return a.SetHandler(c)
	})

	return ContainerAnchorIterator{recv: h, fut: fut.Future, release: release}, nil
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
	return ContainerAnchor{fut: fut, release: release}, nil
}

func (ca ContainerAnchor) Set(ctx context.Context, data []byte) error

func (ca ContainerAnchor) Get(ctx context.Context) ([]byte, error)

type ContainerAnchorIterator struct {
	recv    <-chan []ContainerAnchor
	fut     *capnp.Future
	release capnp.ReleaseFunc

	head ContainerAnchor
	tail []ContainerAnchor
}

func (cai ContainerAnchorIterator) Next(ctx context.Context) error {
	if len(cai.tail) == 0 {
		if err := cai.nextBatch(ctx); err != nil {
			return err
		}
	}

	if len(cai.tail) > 0 {
		cai.head, cai.tail = cai.tail[0], cai.tail[1:]
	}

	return nil
}

func (cai ContainerAnchorIterator) nextBatch(ctx context.Context) (err error) {
	var ok bool
	select {
	case cai.tail, ok = <-cai.recv:
		if !ok {
			_, err = cai.fut.Struct()
		}

	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

func (cai ContainerAnchorIterator) Finish() {
	cai.release()
}

func (cai ContainerAnchorIterator) Anchor() Anchor {
	return cai.head // TODO
}

type handler chan []ContainerAnchor

func (h handler) Shutdown() { close(h) }

func (h handler) Handle(ctx context.Context, call api.Anchor_Handler_handle) error {
	anchors, err := loadBatch(call.Args())
	if err != nil || len(anchors) == 0 { // defensive
		return err
	}

	select {
	case h <- anchors:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func loadBatch(args api.Anchor_Handler_handle_Params) ([]ContainerAnchor, error) {
	capAnchors, err := args.Anchors()
	if err != nil {
		return nil, err
	}

	batch := make([]ContainerAnchor, capAnchors.Len())
	for i := range batch {
		ptr, err := capAnchors.At(i)
		if err != nil {
			return nil, err
		}
		client := api.Container{ptr.Interface().Client()}
		batch[i] = ContainerAnchor{client: client}
	}
	return batch, nil
}
