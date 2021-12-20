package start

import (
	"context"
	"io"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"

	"go.uber.org/fx"

	serviceutil "github.com/wetware/ww/internal/util/service"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/server"
)

var logger = log.New()

var flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "listen",
		Aliases: []string{"a"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/2020/quic",
			"/ip6/::0/udp/2020/quic"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap discovery addr (cidr url)",
		Value:   "tcp://127.0.0.1:8822/24", // TODO:  this should default to mudp
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.StringSliceFlag{
		Name:    "relay",
		Usage:   "pubsub topics to relay",
		EnvVars: []string{"WW_RELAY"},
	},
}

// SetLogger assigns the global logger for this command module.
// It has no effect after the Command().Action has begun executing.
func SetLogger(log log.Logger) { logger = log }

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Flags:  flags,
		Action: run(),
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			node server.Node
			app  = fx.New(fx.NopLogger,
				fx.Populate(&node),
				bind(c))
		)

		if err := start(c, app); err != nil {
			return err
		}

		logger.With(node).Info("wetware loaded")
		<-app.Done() // TODO:  process OS signals in a loop here.

		return shutdown(app)
	}
}

func bind(c *cli.Context) fx.Option {
	return fx.Options(
		runtime.Bind(),
		fx.Supply(c),
		fx.Provide(
			logging,
			supervisor,
			localhost,
			node),
		fx.Invoke(relayTopics))
}

func relayTopics(c *cli.Context, log log.Logger, node server.Node, lx fx.Lifecycle) {
	if topics := c.StringSlice("relay"); len(topics) > 0 {
		for _, topic := range c.StringSlice("relay") {
			lx.Append(newRelayHook(log, node.PubSub(), topic))
		}

		log.WithField("topics", topics).Info("relaying topics")
	}
}

func newRelayHook(log log.Logger, ps server.PubSub, topic string) fx.Hook {
	var (
		t      *pubsub.Topic
		cancel pubsub.RelayCancelFunc
	)

	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			if t, err = ps.Join(topic); err != nil {
				return
			}

			if cancel, err = t.Relay(); err != nil {
				return
			}

			return
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			return t.Close()
		},
	}
}

//
// Dependency declarations
//

func logging() log.Logger {
	return logger
}

func supervisor() *suture.Supervisor {
	return suture.New("runtime", suture.Spec{
		EventHook: serviceutil.NewEventHook(logger, "runtime"),
	})
}

func localhost(c *cli.Context, lx fx.Lifecycle) (host.Host, error) {
	h, err := libp2p.New(c.Context,
		libp2p.NoTransports,
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.ListenAddrStrings(c.StringSlice("listen")...))
	if err == nil {
		lx.Append(closer(h))
	}

	return h, err
}

type serverConfig struct {
	fx.In

	Host      host.Host
	PubSub    *pubsub.PubSub
	Lifecycle fx.Lifecycle
}

func node(c *cli.Context, config serverConfig) (server.Node, error) {
	n, err := server.New(config.Host, config.PubSub,
		server.WithLogger(logger),
		server.WithNamespace(c.String("ns")))

	if err == nil {
		config.Lifecycle.Append(closer(n))
	}

	return n, err
}

func start(c *cli.Context, app *fx.App) error {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*15)
	defer cancel()

	return app.Start(ctx)
}

func shutdown(app *fx.App) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err = app.Stop(ctx); err == context.Canceled {
		err = nil
	}

	return
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
