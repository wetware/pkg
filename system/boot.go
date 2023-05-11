package system

import (
	"io"
	"path"
	"runtime"

	"github.com/lthibault/log"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/discovery"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/boot/socket"
	bootutil "github.com/wetware/casm/pkg/boot/util"
)

type BootConfig struct {
	fx.In

	Log         log.Logger
	Vat         casm.Vat
	BootPeers   boot.StaticAddrs `optional:"true"`
	BootService ma.Multiaddr     `optional:"true"`
}

func DialBoot(config BootConfig, lx fx.Lifecycle) (discovery.Discoverer, error) {
	if len(config.BootPeers) > 0 {
		return config.BootPeers, nil
	}

	if config.BootService == nil {
		config.BootService = ma.StringCast(bootstrapAddr())
	}

	d, err := bootutil.Dial(config.Vat.Host, config.BootService,
		socket.WithLogger(config.Log),
		socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	if c, ok := d.(io.Closer); err == nil && ok {
		lx.Append(fx.StopHook(c.Close))
	}

	return d, err
}

func ListenBoot(config BootConfig, lx fx.Lifecycle) (discovery.Discovery, error) {
	if len(config.BootPeers) > 0 {
		return config.BootPeers, nil
	}

	if config.BootService == nil {
		config.BootService = ma.StringCast(bootstrapAddr())
	}

	d, err := bootutil.Listen(config.Vat.Host, config.BootService,
		socket.WithLogger(config.Log),
		socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	if c, ok := d.(io.Closer); err == nil && ok {
		lx.Append(fx.StopHook(c.Close))
	}

	return d, err
}

func bootstrapAddr() string {
	return path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())
}

func loopback() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		return "lo"
	}
}
