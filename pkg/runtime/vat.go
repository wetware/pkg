package runtime

import (
	"crypto/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"

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

func newVat(mc metrics.Client, flag Flags, lx fx.Lifecycle, f casm.HostFactory) (casm.Vat, error) {
	vat, err := casm.New(flag.String("ns"), f)
	if err == nil {
		lx.Append(closer(vat.Host))
		vat.Metrics = mc.WithPrefix("vat")
	}

	return vat, err
}

func newED25519() (crypto.PrivKey, error) {
	priv, _, err := crypto.GenerateKeyPairWithReader(
		crypto.Ed25519,
		2048,
		rand.Reader)
	return priv, err
}
