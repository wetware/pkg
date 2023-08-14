package csp

import (
	"context"
	"errors"

	capnp "capnproto.org/go/capnp/v3"

	"github.com/ipfs/go-cid"
	api "github.com/wetware/pkg/api/process"
)

// BootContext implements api.BootContext.
type BootContext struct {
	pid  uint32
	cid  cid.Cid
	args capnp.TextList
	caps capnp.PointerList
}

func NewBootContext() *BootContext {
	b := &BootContext{}
	b.Empty()
	return b
}

func (b *BootContext) Empty() *BootContext {
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	b.args, _ = capnp.NewTextList(seg, 0)

	_, seg, _ = capnp.NewMessage(capnp.SingleSegment(nil))
	b.caps, _ = capnp.NewPointerList(seg, 0)

	return b
}

func (b *BootContext) Pid(ctx context.Context, call api.BootContext_pid) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	res.SetPid(b.pid)
	return nil
}

func (b *BootContext) Cid(ctx context.Context, call api.BootContext_cid) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetCid(b.cid.Bytes())
}

func (b *BootContext) Args(ctx context.Context, call api.BootContext_args) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetArgs(b.args)
}

func (b *BootContext) Caps(ctx context.Context, call api.BootContext_caps) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetCaps(b.caps)
}

func (b *BootContext) SetPid(ctx context.Context, call api.BootContext_setPid) error {
	b.pid = call.Args().Pid()
	return nil
}

func (b *BootContext) SetCid(ctx context.Context, call api.BootContext_setCid) error {
	raw, err := call.Args().Cid()
	if err != nil {
		return nil
	}

	_, b.cid, err = cid.CidFromBytes(raw)
	return err
}

func (b *BootContext) WithPid(pid uint32) *BootContext {
	b.pid = pid
	return b
}

func (b *BootContext) WithCid(cid cid.Cid) *BootContext {
	b.cid = cid
	return b
}

func (b *BootContext) WithArgs(args ...string) *BootContext {
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	b.args, _ = capnp.NewTextList(seg, int32(len(args)))

	for i := 0; i < len(args); i++ {
		b.args.Set(i, args[i])
	}

	return b
}

func (b *BootContext) WithRawArgs(args capnp.TextList) *BootContext {
	b.args = args
	return b
}

func (b *BootContext) WithCaps(caps ...capnp.Client) (*BootContext, error) {
	// The caps need to be copied because the original capabilities might be
	// released before the contents are used.
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	b.caps, err = capnp.NewPointerList(seg, int32(len(caps)))
	if err != nil {
		return b, err
	}

	// Continue with the pointer list.
	for i := 0; i < len(caps); i++ {
		_, pSeg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
		if err = b.caps.Set(i, caps[i].EncodeAsPtr(pSeg)); err != nil {
			return b, err
		}
	}

	return b, nil
}

func (b *BootContext) WithRawCaps(caps capnp.PointerList) *BootContext {
	b.caps = caps
	return b
}

func (b *BootContext) Cap() api.BootContext {
	return api.BootContext_ServerToClient(b)
}

// BootCtx is a wrapper for api.BootContext RPCs.
type BootCtx api.BootContext

func (b BootCtx) Pid(ctx context.Context) (uint32, error) {
	f, release := api.BootContext(b).Pid(ctx, nil)
	defer release()
	<-f.Done()

	res, err := f.Struct()
	if err != nil {
		return 0, err
	}
	return res.Pid(), nil
}

func (b BootCtx) Cid(ctx context.Context) (cid.Cid, error) {
	f, release := api.BootContext(b).Cid(ctx, nil)
	defer release()
	<-f.Done()

	res, err := f.Struct()
	if err != nil {
		return cid.Cid{}, err
	}

	raw, err := res.Cid()
	if err != nil {
		return cid.Cid{}, err
	}

	_, cid, err := cid.CidFromBytes(raw)
	return cid, err
}

func (b BootCtx) Args(ctx context.Context) ([]string, error) {
	f, release := api.BootContext(b).Args(ctx, nil)
	defer release()
	<-f.Done()

	res, err := f.Struct()
	if err != nil {
		return nil, err
	}

	tl, err := res.Args()
	if err != nil {
		return nil, err
	}

	args := make([]string, tl.Len())
	for i := 0; i < len(args); i++ {
		args[i], err = tl.At(i)
		if err != nil {
			return nil, err
		}
	}
	return args, nil
}

func (b BootCtx) Caps(ctx context.Context) ([]capnp.Client, error) {
	f, release := api.BootContext(b).Caps(ctx, nil)
	defer release()
	<-f.Done()

	res, err := f.Struct()
	if err != nil {
		return nil, err
	}
	cl, err := res.Caps()
	if err != nil {
		return nil, err
	}

	caps := make([]capnp.Client, cl.Len())

	for i := 0; i < cl.Len(); i++ {
		capPtr, err := cl.At(i)
		if err != nil {
			return nil, err
		}
		var cap capnp.Client
		cap = cap.DecodeFromPtr(capPtr)
		if !cap.IsValid() {
			return nil, errors.New("could not decode cap from pointer")
		}
		caps[i] = cap
	}
	return caps, nil
}

func (b BootCtx) SetPid(ctx context.Context, pid uint32) error {
	f, release := api.BootContext(b).SetPid(ctx, func(bc api.BootContext_setPid_Params) error {
		bc.SetPid(pid)
		return nil
	})
	defer release()
	<-f.Done()

	_, err := f.Struct()
	return err
}

func (b BootCtx) SetCid(ctx context.Context, cid cid.Cid) error {
	f, release := api.BootContext(b).SetCid(ctx, func(bc api.BootContext_setCid_Params) error {
		return bc.SetCid(cid.Bytes())
	})
	defer release()
	<-f.Done()

	_, err := f.Struct()
	return err
}
