package runtime

import (
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
	// TODO:  this should use lx.OnStart to benefit from the start timeout.
	return d.Dial(env.Context())
}

func bootClient(lx fx.Lifecycle, n *client.Node) {
	lx.Append(bootstrapper(n))
}

func clientDefaults(opt []Option) []Option {
	return append([]Option{
		WithHostConfig(casm.Client),
	}, opt...)
}