package statsdutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/metrics"
	"gopkg.in/alexcesaro/statsd.v2"
)

const (
	sampleTick = time.Minute
)

func NewBandwidthCounter(s *statsd.Client) (b *metrics.BandwidthCounter, stop func()) {

	b = metrics.NewBandwidthCounter()
	s.Clone(
		statsd.SampleRate(.1), // send 10% of metrics
		statsd.Prefix("libp2p.host.bandwidth."))

	ticker := time.NewTicker(sampleTick) // 1440 samples/day base-rate
	go func() {
		stat := b.GetBandwidthTotals()
		s.Gauge("rate.in", stat.RateIn)
		s.Gauge("rate.out", stat.RateOut)

		for range ticker.C {
			for proto, stat := range b.GetBandwidthByProtocol() {
				s.Gauge(fmt.Sprintf("%s.rate.in", proto), stat.RateIn)
				s.Gauge(fmt.Sprintf("%s.rate.out", proto), stat.RateOut)
			}
		}
	}()

	return b, ticker.Stop
}

type MetricsProvider interface {
	Metrics() map[string]interface{}
}

type WwMetricsReporter struct {
	providers   []MetricsProvider
	stats       *statsd.Client
	newProvider chan MetricsProvider
}

func NewWwMetricsReporter(stats *statsd.Client) *WwMetricsReporter {
	return &WwMetricsReporter{providers: make([]MetricsProvider, 0), stats: stats, newProvider: make(chan MetricsProvider)}
}

func (m *WwMetricsReporter) Run(ctx context.Context) error {
	ticker := time.NewTicker(sampleTick)
	for {
		select {
		case <-ticker.C:
			m.report()
		case p := <-m.newProvider:
			m.providers = append(m.providers, p)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *WwMetricsReporter) Add(p MetricsProvider) {
	m.newProvider <- p
}

func (m *WwMetricsReporter) NewStore() *MetricStore {
	store := MetricStore{store: make(map[string]interface{})}
	m.newProvider <- &store
	return &store
}

func (m *WwMetricsReporter) report() {
	for _, provider := range m.providers {
		for name, value := range provider.Metrics() {
			m.stats.Gauge(name, value)
		}
	}
}

type MetricStore struct {
	mu    sync.Mutex
	store map[string]interface{}
}

func (m *MetricStore) Add(key string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	num, ok := m.store[key].(int)
	if ok {
		m.store[key] = num + value
	} else {
		m.store[key] = value
	}
}

func (m *MetricStore) Metrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := make(map[string]interface{})
	for key, value := range m.store {
		metrics[key] = value
	}
	return metrics
}
