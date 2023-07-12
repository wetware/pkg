package server

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"

	api "github.com/wetware/ww/api/process"
)

// anyInbox represents any implementation of the capability
type anyIbox interface {
	Open(context.Context, api.Inbox_open) error
}

// decodedInbox holds unencoded capabilities until and encodes them when opened
type decodedInbox struct {
	Content []capnp.Client
}

// encodedInbox holds encoded capabilities and returns them as-is when opened
type encodedInbox struct {
	Content capnp.PointerList
}

func newDecodedInbox(content ...capnp.Client) decodedInbox {
	return decodedInbox{
		Content: content,
	}
}

func newEncodedInbox(content capnp.PointerList) (encodedInbox, error) {
	// the content needs to be copied because the original capability might be
	// released before the contents are used.
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	pl, err := capnp.NewPointerList(seg, int32(content.Len()))
	if err != nil {
		return encodedInbox{}, err
	}

	for i := 0; i < content.Len(); i++ {
		ptr, err := content.At(i)
		if err != nil {
			return encodedInbox{}, err
		}
		_, pSeg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
		var client capnp.Client
		client = client.DecodeFromPtr(ptr)
		if err = pl.Set(i, client.EncodeAsPtr(pSeg)); err != nil {
			return encodedInbox{}, err
		}
	}

	return encodedInbox{
		Content: pl,
	}, nil
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
	// FIXME mikel the error is here
	return res.SetContent(ei.Content)
}
