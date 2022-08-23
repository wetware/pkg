package runtime

import (
	"crypto/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"

	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
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

func (c Config) newHostFactory(env Env, cfg hostFactoryConfig) casm.HostFactory {
	return c.newHost(append([]libp2p.Option{
		libp2p.ListenAddrStrings(env.StringSlice("listen")...),
		// libp2p.BandwidthReporter(cfg.Metrics),
		libp2p.Identity(cfg.Priv),
	}, c.hostOpt...)...)
}

func newVat(env Env, lx fx.Lifecycle, f casm.HostFactory) (casm.Vat, error) {
	vat, err := casm.New(env.String("ns"), f)
	if err == nil {
		lx.Append(closer(vat.Host))
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
