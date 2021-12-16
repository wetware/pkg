package start

import (
	"context"
	"io"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"

	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/internal/runtime"
	serviceutil "github.com/wetware/ww/internal/util/service"
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
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
}

// SetLogger assigns the global logger for this command module.
// It has no effect after the Command().Action has begune executing.
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
		app := fx.New(fx.NopLogger, bind(c))
		if err := app.Start(c.Context); err != nil {
			return err
		}

		<-c.Done() // TODO:  process OS signals in a loop here.

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
			routing,
			node))
}

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

func routing(c *cli.Context, h host.Host) (*dual.DHT, error) {
	return dual.New(c.Context, h,
		dual.LanDHTOption(dht.Mode(dht.ModeServer)),
		dual.WanDHTOption(dht.Mode(dht.ModeAuto)))
}

func node(c *cli.Context, h host.Host, n *cluster.Node) (server.Node, error) {
	return server.New(h, n,
		server.WithLogger(logger),
		server.WithNamespace(c.String("ns")))
}

func shutdown(app *fx.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	return app.Stop(ctx)
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
