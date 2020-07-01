package service

// type startBeaconParams struct {
// 	fx.In

// 	Host host.Host
// 	Boot discover.Protocol
// }

// func startBeacon(ctx context.Context, ps startBeaconParams, lx fx.Lifecycle) error {
// 	bs := beaconService{
// 		Service: ps.Host,
// 		Beacon:  ps.Boot,
// 	}

// 	lx.Append(fx.Hook{
// 		OnStart: func(ctx context.Context) error {
// 			ps.Log.WithFields(bs.Loggable()).Debug("starting service")
// 			return bs.Start(ctx)
// 		},
// 		OnStop: func(context.Context) error {
// 			ps.Log.WithFields(bs.Loggable()).Debug("stopping service")
// 			return bs.Stop(ctx)
// 		},
// 	})

// 	return nil
// }

// type beaconService struct {
// 	Beacon  discover.Beacon
// 	Service interface {
// 		Addrs() []multiaddr.Multiaddr
// 		discover.Service
// 	}
// }

// func (beaconService) Loggable() map[string]interface{} {
// 	return map[string]interface{}{"service": "beacon"}
// }

// func (b beaconService) Start(context.Context) error {
// 	// We must wait until the libp2p.Host is listening before
// 	// advertising our listen addresses.  If you encounter this error,
// 	// try starting the beaconService later.
// 	if len(b.Service.Addrs()) == 0 {
// 		return errors.New("start beacon: host is not listening")
// 	}

// 	if err := b.Beacon.Start(b.Service); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (b beaconService) Stop(context.Context) error {
// 	return b.Beacon.Close()
// }
