package runtime

import (
	casm "github.com/wetware/casm/pkg"
	"go.uber.org/fx"
)

// Client declares dependencies for a *client.Node.
func Client(opt ...Option) fx.Option {
	var c Config
	for _, option := range clientDefaults(opt) {
		option(&c)
	}

	return fx.Module("client",
		c.Vat(),
		c.System(),
		c.ClientBootstrap(),
		//fx.Provide(newClientNode)
	)
}

// func newClientNode(env Env, d client.Dialer) (*client.Node, error) {
// 	ctx, cancel := context.WithTimeout(env.Context(), env.Duration("timeout"))
// 	defer cancel()

// 	return d.Dial(ctx)
// }

func clientDefaults(opt []Option) []Option {
	return append([]Option{
		WithHostConfig(casm.Client),
	}, opt...)
}
