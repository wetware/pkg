package client

import (
	"context"
	"runtime"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	syncutil "github.com/lthibault/util/sync"
	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

type anchor struct{ api.Anchor }

func (a *anchor) Ls(ctx context.Context) ww.Iterator {
	res, err := a.Anchor.Ls(ctx, func(p api.Anchor_ls_Params) error {
		return nil
	}).Struct()
	if err != nil {
		return errIter(err)
	}

	cs, err := res.Children()
	if err != nil {
		return errIter(err)
	}

	return newAnchorIterator(cs)
}

func (a *anchor) Walk(ctx context.Context, path []string) (ww.Anchor, error) {
	res, err := a.Anchor.Walk(ctx, func(param api.Anchor_walk_Params) error {
		return param.SetPath(anchorpath.Join(path...))
	}).Struct()
	if err != nil {
		return nil, err
	}

	// TODO(runtime) runtime.SetFinalizer.  This should probably happen in a
	// 		newRemoteAnchor function or something.
	return &anchor{Anchor: res.Anchor()}, nil
}

type hostAnchor struct {
	anchor
	conn *rpc.Conn
}

func newHostAnchor() *hostAnchor {
	a := new(hostAnchor)

	runtime.SetFinalizer(a, func(ha *hostAnchor) {
		a.Close()
	})

	return a
}

func (ha *hostAnchor) Close() error {
	return ha.conn.Close()
}

func (ha *hostAnchor) Fail(err error) {
	ha.Client = capnp.ErrorClient(err)
}

func (ha *hostAnchor) HandleRPC(ctx context.Context, s network.Stream) error {
	// if the client is already set, it is because an error was encountered.
	if ha.Client != nil {
		// the client is guaranteed to be a capnp.errClient.
		// recover it's error.
		return ha.Client.Call(nil).PipelineClose(nil)
	}

	ha.conn = rpc.NewConn(rpc.StreamTransport(s))
	ha.Client = ha.conn.Bootstrap(ctx)
	return nil
}

func (ha *hostAnchor) Ls(ctx context.Context) ww.Iterator {
	return refIterator{
		Iterator: ha.anchor.Ls(ctx),
		ref:      ha,
	}
}

func (ha *hostAnchor) Walk(ctx context.Context, path []string) (ww.Anchor, error) {
	a, err := ha.anchor.Walk(ctx, path)
	return refAnchor{
		Anchor: a,
		ref:    ha,
	}, err
}

// lazyAnchor is effectively a hostAnchor that has not yet been initalized.
// This is needed because ww.Iterator.Anchor() takes no argument, yet a context is
// needed in order to dial out to the remote host.  As such, we defer dialing until a
// call to one of lazyAnchor's methods is made.
type lazyAnchor struct {
	sess session
	*hostAnchor

	flag syncutil.Flag
	mu   sync.Mutex
}

// ensureConnection is effectively a specialized implementation of sync.Once.Do that
// ensures exactly one connection to a remote host is dialed.  If a connection attempt
// succeeds, ensureConnection returns nil, and subsequent calls are nops.
//
// For the avoidance of doubt:  calling ensureConnection after it has returned a non-nil
// error is legal, and will attempt to connect to the remote host.
//
// ensureConnection is thread-safe.
func (la *lazyAnchor) ensureConnection(ctx context.Context) (err error) {
	if la.flag.Bool() {
		// a previous call completed successfully
		return
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	// we hold the lock, so we can access fields directly.
	if la.flag != 0 {
		// a concurrent call completed successfully while
		// we were waiting for the lock
		return
	}

	ha := newHostAnchor()
	if err = la.sess.Call(ctx, ww.AnchorProtocol, ha); err == nil {
		la.hostAnchor = ha
		la.flag.Set()
	}

	return
}

func (la *lazyAnchor) Ls(ctx context.Context) ww.Iterator {
	if err := la.ensureConnection(ctx); err != nil {
		return errIter(err)
	}

	return la.hostAnchor.Ls(ctx)
}

func (la *lazyAnchor) Walk(ctx context.Context, path []string) (_ ww.Anchor, err error) {
	if err := la.ensureConnection(ctx); err != nil {
		return nil, errIter(err)
	}

	return la.hostAnchor.Walk(ctx, path)
}

// refAnchor holds a reference to a hostAnchor, preventing the latter from being
// garbage-collected.  See call to runtime.SetFinalizer in newHostAnchor.
type refAnchor struct {
	ww.Anchor
	ref interface{}
}

func (ra refAnchor) Ls(ctx context.Context) ww.Iterator {
	return refIterator{
		Iterator: ra.Anchor.Ls(ctx),
		ref:      ra.ref,
	}
}

func (ra refAnchor) Walk(ctx context.Context, path []string) (ww.Anchor, error) {
	a, err := ra.Anchor.Walk(ctx, path)
	return refAnchor{
		Anchor: a,
		ref:    ra.ref,
	}, err
}
