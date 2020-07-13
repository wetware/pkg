package service

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/lthibault/wetware/pkg/runtime"
)

// EvtTimestep represents an increment in the runtime's logical clock.
// To ensure reproducible tests, Services should consume EvtTimestep instead of
// maintaining internal clocks, timers or tickers.
type EvtTimestep struct {
	Time  time.Time
	Delta time.Duration
}

// Ticker emits a timestep for consumption by downstream services.
func Ticker(bus event.Bus, step time.Duration) ProviderFunc {
	return func() (runtime.Service, error) {
		e, err := bus.Emitter(new(EvtTimestep))
		if err != nil {
			return nil, err
		}

		return &ticker{
			step: step,
			cq:   make(chan struct{}),
			errs: make(chan error, 1),
			e:    e,
		}, nil
	}
}

type ticker struct {
	cq   chan struct{}
	errs chan error

	step time.Duration
	t    *time.Ticker
	e    event.Emitter
}

func (t ticker) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service":  "ticker",
		"timestep": t.step,
	}
}

func (t ticker) Errors() <-chan error {
	return t.errs
}

func (t *ticker) Start(context.Context) error {
	t.t = time.NewTicker(t.step)
	go t.loop()
	return nil
}

func (t ticker) loop() {
	defer close(t.errs)

	var ts EvtTimestep
	for tick := range t.t.C {
		ts.Delta = tick.Sub(ts.Time)
		ts.Time = tick
		t.emit(ts)
	}
}

func (t ticker) Stop(context.Context) error {
	t.t.Stop()
	return t.e.Close()
}

func (t ticker) emit(ev EvtTimestep) {
	t.raise(t.e.Emit(ev))
}

func (t ticker) raise(err error) {
	if err == nil {
		return
	}

	select {
	case t.errs <- err:
	case <-t.cq:
	}
}
