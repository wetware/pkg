package runtime

import (
	"context"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/ww/pkg/client"
	"go.uber.org/fx"
)

func Client(opt ...Option) fx.Option {
	var c Config
	for _, option := range clientDefaults(opt) {
		option(&c)
	}

	return fx.Module("client",
		c.Vat(),
		c.System(),
		c.ClientBootstrap(),
		fx.Provide(newClientNode),
		fx.Invoke(bootClient))
}

func newClientNode(env Env, d client.Dialer) (*client.Node, error) {
	ctx, cancel := context.WithTimeout(env.Context(), env.Duration("timeout"))
	defer cancel()

	return d.Dial(ctx)
}

func bootClient(lx fx.Lifecycle, n *client.Node) {
	lx.Append(bootstrapper(n))
}

func clientDefaults(opt []Option) []Option {
	return append([]Option{
		WithHostConfig(casm.Client),
	}, opt...)
}
