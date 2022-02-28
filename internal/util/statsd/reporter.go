package statsdutil

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/metrics"
	"gopkg.in/alexcesaro/statsd.v2"
)

func NewBandwidthCounter(s *statsd.Client) (b *metrics.BandwidthCounter, stop func()) {

	b = metrics.NewBandwidthCounter()
	s.Clone(
		statsd.SampleRate(.1), // send 10% of metrics
		statsd.Prefix("libp2p.host.bandwidth."))

	ticker := time.NewTicker(time.Minute) // 1440 samples/day base-rate
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
