package start

import (
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/cluster/pulse"
	"github.com/wetware/ww/pkg/boot"
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

func newBootStrategy(c *cli.Context, log log.Logger) bootServices {
	const (
		addr = "0.0.0.0:8822"
		port = 8822
		cidr = "255.255.255.0/24"
	)

	var b = boot.Beacon{
		Logger: log.
			WithField("addr", addr),
		Addr: addr,
	}

	var s = boot.Scanner{
		Logger: log.
			WithField("port", 8822).
			WithField("cidr", cidr),
		Port: port,
		CIDR: cidr,
	}

	return bootServices{
		Beacon:     &b,
		Advertiser: &b,
		Discoverer: &s,
	}
}
