package client

import (
	"context"

	"go.uber.org/fx"

	p2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
)

type hostParams struct {
	fx.In

	Ctx      context.Context
	Cfg      *Config
	Discover struct{ Discover }
}

func newHost(lx fx.Lifecycle, p hostParams) host.Host {
	var h struct{ host.Host }

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			h.Host, err = p2p.New(p.Ctx,
				p.Cfg.maybePSK(),
				p2p.Ping(false),
				p2p.NoListenAddrs, // also disables relay
				p2p.UserAgent("ww client"))
			return
		},
		OnStop: func(context.Context) error {
			return h.Close()
		},
	})

	return &h
}
