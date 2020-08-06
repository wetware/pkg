//go:generate mockgen -destination ../../../internal/test/mock/pkg/runtime/service/mock_service.go github.com/wetware/ww/pkg/runtime/service Publisher

// Package service contains runtime service definitions
package service

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/lthibault/jitterbug"
	"github.com/pkg/errors"
	"github.com/wetware/ww/pkg/internal/p2p"
	"github.com/wetware/ww/pkg/runtime"
)

// ProviderFunc satisfies runtime.ServiceFactory.
type ProviderFunc func() (runtime.Service, error)

// Service initializes a new runtime service.
func (f ProviderFunc) Service() (runtime.Service, error) {
	return f()
}

/*
	Internal utilities
*/

func waitNetworkReady(ctx context.Context, bus event.Bus) error {
	sub, err := bus.Subscribe(new(p2p.EvtNetworkReady))
	if err != nil {
		return err
	}
	defer sub.Close()

	select {
	case <-sub.Out():
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "wait network ready")
	}
}

type scheduler struct {
	d, remaining time.Duration
	j            jitterbug.Jitter
}

func newScheduler(d time.Duration, j jitterbug.Jitter) *scheduler {
	s := &scheduler{d: d, j: j}
	s.Reset()
	return s
}

func (s *scheduler) Advance(d time.Duration) bool {
	s.remaining -= d
	return s.remaining <= 0
}

func (s *scheduler) Reset() {
	s.remaining = s.j.Jitter(s.d)
}

func startBackground(fs ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(fs))
	defer wg.Wait()

	for _, f := range fs {
		go func(f func()) {
			wg.Done()
			f()
		}(f)
	}
}
