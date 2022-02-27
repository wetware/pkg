package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/cluster"
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

func (ca ContainerAnchor) Ls(ctx context.Context, path []string) (AnchorIterator, error) {
	return newIterator(ctx, ca.fut.Anchor(), path)
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
