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
	reportTick = time.Minute
)

func NewBandwidthCounter(s *statsd.Client) (b *metrics.BandwidthCounter, stop func()) {

	b = metrics.NewBandwidthCounter()
	s.Clone(
		statsd.SampleRate(.1), // send 10% of metrics
		statsd.Prefix("libp2p.host.bandwidth."))

	ticker := time.NewTicker(reportTick) // 1440 samples/day base-rate
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
	GaugeMetrics() map[string]interface{}
	CountMetrics() map[string]interface{}
}

type MetricsReporter struct {
	providers   []MetricsProvider
	stats       *statsd.Client
	newProvider chan MetricsProvider
}

func NewMetricsReporter(stats *statsd.Client) *MetricsReporter {
	return &MetricsReporter{providers: make([]MetricsProvider, 0), stats: stats, newProvider: make(chan MetricsProvider)}
}

func (m *MetricsReporter) Run(ctx context.Context) error {
	ticker := time.NewTicker(reportTick)
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

func (m *MetricsReporter) Add(p MetricsProvider) {
	m.newProvider <- p
}

func (m *MetricsReporter) NewStore() *MetricStore {
	store := NewMetricStore()
	m.newProvider <- store
	return store
}

func (m *MetricsReporter) report() {
	for _, provider := range m.providers {
		if metrics := provider.GaugeMetrics(); metrics != nil {
			for name, value := range metrics {
				m.stats.Gauge(name, value)
			}
		}

		if metrics := provider.CountMetrics(); metrics != nil {
			for name, value := range metrics {
				m.stats.Count(name, value)
			}
		}
	}
}

type MetricStore struct {
	mu         sync.Mutex
	gaugeStore map[string]interface{}
	countStore map[string]interface{}
}

func NewMetricStore() *MetricStore {
	return &MetricStore{gaugeStore: make(map[string]interface{}), countStore: make(map[string]interface{})}
}

func (m *MetricStore) GaugeAdd(key string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	num, ok := m.gaugeStore[key].(int)
	if ok {
		m.gaugeStore[key] = num + value
	} else {
		m.gaugeStore[key] = value
	}
}

func (m *MetricStore) GaugeSet(key string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gaugeStore[key] = value
}

func (m *MetricStore) CountAdd(key string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	num, ok := m.countStore[key].(int)
	if ok {
		m.countStore[key] = num + value
	} else {
		m.countStore[key] = value
	}
}

func (m *MetricStore) CountSet(key string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.countStore[key] = value
}

func (m *MetricStore) GaugeMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := make(map[string]interface{})
	for key, value := range m.gaugeStore {
		metrics[key] = value
	}
	return metrics
}

func (m *MetricStore) CountMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := make(map[string]interface{})
	for key, value := range m.countStore {
		metrics[key] = value
	}

	m.countStore = make(map[string]interface{}) // reset values

	return metrics
}
