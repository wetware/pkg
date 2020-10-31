package internal

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/lthibault/jitterbug"
	"github.com/pkg/errors"
	"github.com/wetware/ww/pkg/internal/p2p"
)

func WaitNetworkReady(ctx context.Context, bus event.Bus) error {
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

type Scheduler struct {
	d, remaining time.Duration
	j            jitterbug.Jitter
}

func NewScheduler(d time.Duration, j jitterbug.Jitter) *Scheduler {
	s := &Scheduler{d: d, j: j}
	s.Reset()
	return s
}

func (s *Scheduler) Advance(d time.Duration) bool {
	s.remaining -= d
	return s.remaining <= 0
}

func (s *Scheduler) Reset() {
	s.remaining = s.j.Jitter(s.d)
}

func StartBackground(fs ...func()) {
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
