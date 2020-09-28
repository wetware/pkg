package service

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/runtime"
)

// EvtTimestep represents an increment in the runtime's logical clock.
// To ensure reproducible tests, Services should consume EvtTimestep instead of
// maintaining internal clocks, timers or tickers.
type EvtTimestep struct {
	Time  time.Time
	Delta time.Duration
}

// Ticker emits a timestep for consumption by downstream services.
func Ticker(log ww.Logger, bus event.Bus, step time.Duration) ProviderFunc {
	return func() (runtime.Service, error) {
		e, err := bus.Emitter(new(EvtTimestep))
		if err != nil {
			return nil, err
		}

		return &ticker{
			log:  log,
			step: step,
			cq:   make(chan struct{}),
			e:    e,
		}, nil
	}
}

type ticker struct {
	log ww.Logger
	cq  chan struct{}

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

func (t *ticker) Start(context.Context) error {
	t.t = time.NewTicker(t.step)
	go t.loop()
	return nil
}

func (t ticker) loop() {

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
	if err := t.e.Emit(ev); err != nil {
		t.log.With(t).WithError(err).Error("failed to emit EvtTimestep")
	}
}
