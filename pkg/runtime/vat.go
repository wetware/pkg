package runtime

import (
	"crypto/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/lthibault/log"

	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/metrics"
)

func (c Config) Vat() fx.Option {
	return fx.Module("vat",
		fx.Provide(
			c.newHostFactory,
			newVat,
			newED25519))
}

type hostFactoryConfig struct {
	fx.In

	// Metrics *host_metrics.BandwidthCounter
	Priv crypto.PrivKey
}

func (c Config) newHostFactory(flag Flags, cfg hostFactoryConfig) casm.HostFactory {
	return c.newHost(append([]libp2p.Option{
		libp2p.ListenAddrStrings(flag.StringSlice("listen")...),
		// libp2p.BandwidthReporter(cfg.Metrics),
		libp2p.Identity(cfg.Priv),
	}, c.hostOpt...)...)
}

type vatConfig struct {
	fx.In

	Flags   Flags
	Log     log.Logger
	Metrics metrics.Client
	NewHost casm.HostFactory
}

func (config vatConfig) New() (vat casm.Vat, err error) {
	if vat.Host, err = config.NewHost(); err == nil {
		vat.NS = config.Flags.String("ns")
		vat.Logger = config.Log
		vat.Metrics = config.Metrics.WithPrefix("vat")
	}

	return
}

func newVat(config vatConfig, lx fx.Lifecycle) (vat casm.Vat, err error) {
	if vat, err = config.New(); err == nil {
		lx.Append(closer(vat.Host))
	}

	return
}

func newED25519() (crypto.PrivKey, error) {
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	return priv, err
}
