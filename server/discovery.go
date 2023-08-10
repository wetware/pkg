package server

import (
	"io"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/boot/socket"
)

func (cfg Config) newBootstrapper(h host.Host) (*bootService, error) {
	var d discovery.Discovery
	var err error
	if len(cfg.Peers) > 0 {
		d, err = boot.NewStaticAddrStrings(cfg.Peers...)
	} else {
		d, err = boot.ListenString(h, cfg.Discover,
			socket.WithLogger(cfg.Logger),
			socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	}

	if err != nil {
		return nil, err
	}

	return &bootService{Discovery: d}, nil
}

type bootService struct{ discovery.Discovery }

func (d bootService) Close() (err error) {
	if c, ok := d.Discovery.(io.Closer); ok {
		err = c.Close()
	}

	return
}
