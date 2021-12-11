package start

import (
	"context"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/cluster/pulse"
	"go.uber.org/fx"
)

// systemHook populates heartbeat messages with system information from the
// operating system.
type systemHook struct{}

func newSystemHook() pulse.Preparer {
	return systemHook{}
}

func (h systemHook) Prepare(pulse.Heartbeat) {
	// TODO:  populate a capnp struct containing metadata for the
	//        local host.  Consider including AWS AR information,
	//        hostname, geolocalization, and a UUID instance id.

	// WARNING:  DO NOT make a syscall each time 'Prepare' is invoked.
	//           Cache results and periodically refresh them.
}

type bootServices struct {
	fx.Out

	Beacon     suture.Service `group:"services"`
	Advertiser discovery.Advertiser
	Discoverer discovery.Discoverer
}

func newBootStrategy(c *cli.Context, log log.Logger, lx fx.Lifecycle) bootServices {
	var b = boot.Beacon{
		Log:  log,
		Addr: "0.0.0.0:8822",
	}

	var s = boot.Scanner{
		Port: 8822,
		CIDR: "255.255.255.0/24",
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return s.Close()
		},
	})

	return bootServices{
		Beacon:     &b,
		Advertiser: &b,
		Discoverer: &s,
	}
}
