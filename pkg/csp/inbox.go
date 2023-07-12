package csp

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"

	api "github.com/wetware/ww/api/process"
)

type anyIbox interface {
	Open(context.Context, api.Inbox_open) error
}

type decodedInbox struct {
	Content []capnp.Client
}

type encodedInbox struct {
	Content capnp.PointerList
}

func newDecodedInbox(content ...capnp.Client) decodedInbox {
	return decodedInbox{
		Content: content,
	}
}

func newEncodedInbox(content capnp.PointerList) encodedInbox {
	return encodedInbox{
		Content: content,
	}
}

func (di decodedInbox) Open(ctx context.Context, call api.Inbox_open) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	cs, err := res.NewContent(int32(len(di.Content)))
	if err != nil {
		return err
	}

	for i, content := range di.Content {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return err
		}
		if err = cs.Set(i, content.EncodeAsPtr(seg)); err != nil {
			return err
		}
	}

	return res.SetContent(cs)
}

func (ei encodedInbox) Open(ctx context.Context, call api.Inbox_open) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	return res.SetContent(ei.Content)
}
