package start

import (
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
)

func newSharedSecret(c *cli.Context) fx.Option {
	if c.String("secret") == "" {
		return fx.Options() // nop
	}

	return fx.Supply(pnet.PSK(c.String("secret")))
}
