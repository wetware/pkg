package runtime

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	log "github.com/lthibault/log/pkg"
	"github.com/lthibault/wetware/pkg/discover"
	"github.com/pkg/errors"
	"go.uber.org/fx"
)

type startBeaconParams struct {
	fx.In

	Log  log.Logger
	Host host.Host
	Boot discover.Protocol
}

func startBeacon(ctx context.Context, ps startBeaconParams, lx fx.Lifecycle) error {
	log := ps.Log.WithField("service", "beacon")

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {

			// We must wait until the libp2p.Host is listening before
			// advertising our listen addresses.  If you encounter this error,
			// try starting the beaconService later.
			if len(ps.Host.Addrs()) == 0 {
				return errors.New("start beacon: host is not listening")
			}

			if err := ps.Boot.Start(ps.Host); err != nil {
				return err
			}

			log.Debug("service started")
			return nil
		},
		OnStop: func(context.Context) error {
			defer log.Debug("service started")
			return ps.Boot.Close()
		},
	})

	return nil
}
