package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	api "github.com/wetware/ww/internal/api/cluster"
	"golang.org/x/sync/semaphore"
)

type Iterator struct {
	path []string

	recv    <-chan []ContainerAnchor
	fut     *capnp.Future
	release capnp.ReleaseFunc

	head ContainerAnchor
	tail []ContainerAnchor

	err error
}

func newIterator(ctx context.Context, anchor api.Anchor, path []string) (*Iterator, error) {
	h := handler{path: path, recv: make(chan []ContainerAnchor, defaultMaxInflight)}

	fut, release := anchor.Ls(ctx, func(a api.Anchor_ls_Params) error {
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
			MaxConcurrentCalls: cap(h.recv),
			AnswerQueueSize:    cap(h.recv),
		})
		return a.SetHandler(c)
	})

	return &Iterator{path: path, recv: h.recv, fut: fut.Future, release: release}, nil
}

func (it *Iterator) Next(ctx context.Context) bool {
	if len(it.tail) == 0 {
		if it.err = it.nextBatch(ctx); it.err != nil {
			return false
		}
	}

	if len(it.tail) > 0 {
		it.head, it.tail = it.tail[0], it.tail[1:]
		return true
	}

	return false
}

func (it *Iterator) nextBatch(ctx context.Context) (err error) {
	var ok bool
	select {
	case it.tail, ok = <-it.recv:
		if !ok {
			_, err = it.fut.Struct()
		}

	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

func (it *Iterator) Finish() {
	it.release()
}

func (it *Iterator) Anchor() Anchor {
	return it.head
}

func (it *Iterator) Err() error {
	return it.err
}

type handler struct {
	path []string
	recv chan []ContainerAnchor
}

func (h handler) Shutdown() { close(h.recv) }

func (h handler) Handle(ctx context.Context, call api.Anchor_Handler_handle) error {
	anchors, err := h.loadBatch(call.Args())
	if err != nil || len(anchors) == 0 { // defensive
		return err
	}

	select {
	case h.recv <- anchors:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h handler) loadBatch(args api.Anchor_Handler_handle_Params) ([]ContainerAnchor, error) {
	capAnchors, err := args.Anchors()
	if err != nil {
		return nil, err
	}

	batch := make([]ContainerAnchor, capAnchors.Len())
	for i := range batch {
		name, err := capAnchors.At(i).Name()
		if err != nil {
			return nil, err
		}
		client := api.Container{Client: capAnchors.At(i).Anchor().Client}
		batch[i] = ContainerAnchor{path: append(h.path, name), client: client}
	}
	return batch, nil
}

type batcher struct {
	h       api.Anchor_Handler
	limiter *semaphore.Weighted
	fs      map[*capnp.Future]capnp.ReleaseFunc // in-flight

	batch []AnchorElement
}

type AnchorElement struct {
	name   string
	anchor api.Anchor
}

func newBatcher(h api.Anchor_Handler) batcher {
	return batcher{
		limiter: semaphore.NewWeighted(defaultMaxInflight),
		h:       h,
		fs:      make(map[*capnp.Future]capnp.ReleaseFunc),
		batch:   make([]AnchorElement, 0, defaultBatchSize),
	}
}

func (b *batcher) Send(ctx context.Context, anchor api.Anchor, name string) error {
	b.batch = append(b.batch, AnchorElement{name: name, anchor: anchor})
	if len(b.batch) == cap(b.batch) {
		return b.Flush(ctx)
	}

	return nil
}

func (b *batcher) Flush(ctx context.Context) error {
	if err := b.limiter.Acquire(ctx, 1); err != nil {
		return err
	}

	f, release := b.h.Handle(ctx, func(a api.Anchor_Handler_handle_Params) error {
		defer func() {
			b.batch = b.batch[:0]
		}()

		anchors, err := a.NewAnchors(int32(len(b.batch)))
		if err != nil {
			return err
		}

		for i, e := range b.batch {
			anchors.At(i).SetName(e.name)
			anchors.At(i).SetAnchor(e.anchor)
		}

		return err
	})
	b.fs[f.Future] = func() {
		delete(b.fs, f.Future)
		release()
		b.limiter.Release(1)
	}

	// release any resolved futures and return their errors, if any
	for f, release := range b.fs {
		select {
		case <-f.Done():
			defer release()
			if _, err := f.Struct(); err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (b *batcher) Wait(ctx context.Context) (err error) {
	if err = b.Flush(ctx); err != nil {
		return
	}

	for f, release := range b.fs {
		// This is a rare case in which the use of 'defer' in
		// a loop is NOT a bug.
		//
		// We iterate over the whole map in order to schedule
		// a deferred call to 'release' for all pending RPC
		// calls.
		//
		// In principle, this is not necessary since resources
		// will be released when the handler for Iter returns.
		// We do it anyway to guard against bugs and/or changes
		// in the capnp API.
		defer release()

		// We want to abort early if any future encounters an
		// error, but as per the previous comment, we also want
		// to defer a call to 'release' for each future.
		if err == nil {
			// We're waiting until all futures resolve, so it's
			// okay to block on any given 'f'.
			select {
			case <-f.Done():
				_, err = f.Struct()

			case <-ctx.Done():
				err = ctx.Err()
			}
		}
	}

	return
}
