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
		statsd.SampleRate(1.), // send 100% of metrics
		statsd.Prefix("libp2p.host.bandwidth."))

	ticker := time.NewTicker(time.Minute) // 1440 samples/day
	go func() {
		stat := b.GetBandwidthTotals()
		s.Gauge("rate.in", stat.RateIn)
		s.Gauge("rate.out", stat.RateOut)
		s.Gauge("total.in", stat.TotalIn)
		s.Gauge("total.out", stat.TotalOut)

		for range ticker.C {
			for id, stat := range b.GetBandwidthByPeer() {
				prefix := fmt.Sprintf("peer.%s.", id)
				s.Gauge(prefix+"rate.in", stat.RateIn)
				s.Gauge(prefix+"rate.out", stat.RateOut)
				s.Gauge(prefix+"total.in", stat.TotalIn)
				s.Gauge(prefix+"total.out", stat.TotalOut)
			}

			for proto, stat := range b.GetBandwidthByProtocol() {
				prefix := fmt.Sprintf("proto.%s.", proto)
				s.Gauge(prefix+"rate.in", stat.RateIn)
				s.Gauge(prefix+"rate.out", stat.RateOut)
				s.Gauge(prefix+"total.in", stat.TotalIn)
				s.Gauge(prefix+"total.out", stat.TotalOut)
			}
		}
	}()

	return b, ticker.Stop
}
