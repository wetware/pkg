package capstore

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/capstore"
)

type CapStore api.CapStore

func (c CapStore) Set(ctx context.Context, id string, cap capnp.Client) error {
	f, release := api.CapStore(c).Set(ctx, func(cs api.CapStore_set_Params) error {
		if err := cs.SetId(id); err != nil {
			return err
		}
		return cs.SetCap(cap)
	})
	defer release()

	<-f.Done()
	_, err := f.Struct()
	return err
}

func (c CapStore) Get(ctx context.Context, id string) (capnp.Client, error) {
	f, release := api.CapStore(c).Get(ctx, func(cs api.CapStore_get_Params) error {
		return cs.SetId(id)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return capnp.Client{}, err
	}

	return res.Cap(), nil
}
