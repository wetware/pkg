package start

import (
	"net"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/packet"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/server"
	"go.uber.org/fx"
)

type bootService struct {
	fx.Out

	Service   suture.Service `group:"services"`
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
		Service:   &strategy.PortListener,
		Bootstrap: &strategy.PortListener,
		Strategy:  &strategy,
	}, nil
}
