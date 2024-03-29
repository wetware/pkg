package capstore_server

import (
	"context"
	"fmt"
	"sync"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/capstore"
	"github.com/wetware/pkg/cap/capstore"
	"github.com/wetware/pkg/util/log"
)

type CapStore struct {
	// TODO limit map size
	*sync.Map
	log.Logger
}

func (c *CapStore) CapStore() capstore.CapStore {
	return capstore.CapStore(api.CapStore_ServerToClient(c))
}

func (c *CapStore) Set(ctx context.Context, call api.CapStore_set) error {
	id, err := call.Args().Id()
	if err != nil {
		return err
	}

	cap := call.Args().Cap()
	c.Logger.Info("set capability", "id", id)

	c.Map.Store(id, cap.AddRef())
	return nil
}

func (c *CapStore) Get(ctx context.Context, call api.CapStore_get) error {
	id, err := call.Args().Id()
	if err != nil {
		return err
	}

	c.Logger.Info("get capability", "id", id)
	v, ok := c.Map.Load(id)
	if !ok {
		return fmt.Errorf("capability with id '%s' not found", id)
	}

	cap, ok := v.(capnp.Client)
	if !ok {
		return fmt.Errorf("capability with id '%s' not found", id)
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetCap(cap.AddRef())
}
