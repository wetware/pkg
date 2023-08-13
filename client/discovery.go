package client

import (
	"io"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/boot/socket"
)

func (cfg DialConfig) newBootstrapper(h host.Host) (*bootService, error) {
	var d discovery.Discoverer
	var err error
	if len(cfg.Peers) > 0 {
		d, err = boot.NewStaticAddrStrings(cfg.Peers...)
	} else {
		d, err = boot.DialString(h, cfg.Discover,
			// socket.WithLogger(cfg.Logger),
			socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	}

	if err != nil {
		return nil, err
	}

	return &bootService{Discoverer: d}, nil
}

type bootService struct{ discovery.Discoverer }

func (d bootService) Close() (err error) {
	if c, ok := d.Discoverer.(io.Closer); ok {
		err = c.Close()
	}

	return
}
