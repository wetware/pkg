package start

import (
	"net"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/cluster/pulse"
	"github.com/wetware/casm/pkg/packet"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/server"
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

type bootService struct {
	fx.Out

	Services  []suture.Service `group:"services,flatten"`
	Bootstrap discovery.Advertiser
	Strategy  server.BootStrategy
}

func newBootStrategy(c *cli.Context, log log.Logger, lx fx.Lifecycle) (bootService, error) {
	const (
		port = 8822
		cidr = "127.0.0.1/24"
	)

	log = log.WithField("port", port)

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return bootService{}, err
	}

	var strategy = server.PortScanStrategy{
		PortListener: boot.PortListener{
			Logger: log,
			Endpoint: packet.Endpoint{
				Addr: &net.UDPAddr{
					IP:   net.IPv4zero, // listen on all interfaces
					Port: port,
				},
			},
		},
		PortKnocker: boot.PortKnocker{
			Logger: log.WithField("cidr", cidr),
			Port:   port,
			Subnet: ipnet,
		},
	}

	return bootService{
		Services: []suture.Service{
			&strategy.PortListener},
		Bootstrap: &strategy.PortListener,
		Strategy:  &strategy,
	}, nil
}
