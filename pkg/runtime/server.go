package runtime

import (
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/ww/pkg/server"
	"go.uber.org/fx"
)

func Server(opt ...Option) fx.Option {
	var c Config
	for _, option := range serverDefaults(opt) {
		option(&c)
	}

	return fx.Module("server",
		c.Vat(),
		c.System(),
		c.PubSub(),
		c.Routing(),
		c.ServerBootstrap(),
		fx.Provide(newServerNode),
		fx.Invoke(bootServer))
}

func serverDefaults(opt []Option) []Option {
	return append([]Option{
		WithHostConfig(casm.Server),
	}, opt...)
}

func newServerNode(j server.Joiner, ps *pubsub.PubSub) (*server.Node, error) {
	// TODO:  this should use lx.OnStart to benefit from the start timeout.
	return j.Join(ps)
}

func bootServer(lx fx.Lifecycle, n *server.Node) {
	lx.Append(fx.Hook{
		OnStart: bootstrap(n),
		OnStop:  onclose(n),
	})
}
