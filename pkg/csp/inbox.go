package csp

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"

	api "github.com/wetware/ww/api/process"
)

// type Inbox api.Inbox

// func (in Inbox) AddRef() Inbox {
// 	return Inbox(capnp.Client(in).AddRef())
// }

// func (in Inbox) Release() {
// 	capnp.Client(in).Release()
// }

type inboxServer struct {
	Content capnp.Client
}

func (is inboxServer) Open(ctx context.Context, call api.Inbox_open) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetContent(is.Content.AddRef())
}
