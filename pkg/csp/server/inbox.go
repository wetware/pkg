package server

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"

	api "github.com/wetware/ww/api/process"
)

// anyContext represents any implementation of the capability
type anyIbox interface {
	Open(context.Context, api.Context_open) error
}

// decodedContext holds unencoded capabilities until and encodes them when opened
type decodedContext struct {
	Content []capnp.Client
}

// encodedContext holds encoded capabilities and returns them as-is when opened
type encodedContext struct {
	Content capnp.PointerList
}

func newDecodedContext(content ...capnp.Client) decodedContext {
	return decodedContext{
		Content: content,
	}
}

func newEncodedContext(content capnp.PointerList, prepend ...capnp.Client) (encodedContext, error) {
	// The content needs to be copied because the original capability might be
	// released before the contents are used.
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	pl, err := capnp.NewPointerList(seg, int32(content.Len()+len(prepend)))
	if err != nil {
		return encodedContext{}, err
	}

	delta := len(prepend)

	// Start by adding al prependable caps.
	for i := 0; i < delta; i++ {
		_, pSeg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
		if err = pl.Set(i, prepend[i].EncodeAsPtr(pSeg)); err != nil {
			return encodedContext{}, err
		}
	}

	// Continue with the pointer list.
	for i := delta; i < content.Len()+delta; i++ {
		ptr, err := content.At(i - delta)
		if err != nil {
			return encodedContext{}, err
		}
		_, pSeg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
		var client capnp.Client
		client = client.DecodeFromPtr(ptr)
		if err = pl.Set(i, client.EncodeAsPtr(pSeg)); err != nil {
			return encodedContext{}, err
		}
	}

	return encodedContext{
		Content: pl,
	}, nil
}

func (di decodedContext) Open(ctx context.Context, call api.Context_open) error {
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

func (ei encodedContext) Open(ctx context.Context, call api.Context_open) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	return res.SetContent(ei.Content)
}
