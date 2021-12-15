package boot

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
)

type Service struct {
	discovery.Advertiser
	discovery.Discoverer
}

type Context struct {
	Strategy ScanStrategy
	Net      Dialer

	once  sync.Once
	scans chan *scanRequest
}

func (s *Context) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	s.once.Do(func() { s.scans = make(chan *scanRequest) })

	opts := &discovery.Options{
		Limit: 1,
	}

	if err := opts.Apply(opt...); err != nil {
		return nil, err
	}

	chout := make(chan peer.AddrInfo, 1)
	select {
	case s.scans <- &scanRequest{opts: opts, chout: chout}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return chout, nil
}

func (s *Context) Serve(ctx context.Context) error {
	s.once.Do(func() { s.scans = make(chan *scanRequest) })

	var (
		errs = make(chan error, 1)
		rec  peer.PeerRecord
	)

	for {
		select {
		case scan := <-s.scans:
			scanner := scan.Bind(s.Net, &rec, s.Strategy)
			go s.run(ctx, func(e error) { errs <- e }, scanner)

		case err := <-errs:
			return err

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Context) run(ctx context.Context, raise func(error), scan func(context.Context) error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		if err := scan(ctx); err == nil {
			break
		}
	}
}

type scanRequest struct {
	opts  *discovery.Options
	chout chan<- peer.AddrInfo
}

func (s *scanRequest) Bind(d Dialer, r record.Record, strategy ScanStrategy) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		_, err := strategy.Scan(ctx, d, r)
		return err
	}
}
