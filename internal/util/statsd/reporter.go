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
		for range ticker.C {
			for id, stat := range b.GetBandwidthByPeer() {
				prefix := fmt.Sprintf("peer.%s.", id)
				s.Gauge(prefix+"rate_in", stat.RateIn)
				s.Gauge(prefix+"rate_out", stat.RateOut)
				s.Gauge(prefix+"total_in", stat.TotalIn)
				s.Gauge(prefix+"total_out", stat.TotalOut)
			}

			for proto, stat := range b.GetBandwidthByProtocol() {
				prefix := fmt.Sprintf("proto.%s.", proto)
				s.Gauge(prefix+"rate_in", stat.RateIn)
				s.Gauge(prefix+"rate_out", stat.RateOut)
				s.Gauge(prefix+"total_in", stat.TotalIn)
				s.Gauge(prefix+"total_out", stat.TotalOut)
			}
		}
	}()

	return b, ticker.Stop
}
