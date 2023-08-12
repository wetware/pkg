package cluster

import (
	"context"
	"log"
	"path"
	"runtime"
	"time"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"golang.org/x/exp/slog"

	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/client"
)

// Logger is used for logging by the RPC system. Each method logs
// messages at a different level, but otherwise has the same semantics:
//
//   - Message is a human-readable description of the log event.
//   - Args is a sequenece of key, value pairs, where the keys must be strings
//     and the values may be any type.
//   - The methods may not block for long periods of time.
//
// This interface is designed such that it is satisfied by *slog.Logger.
type Logger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
	With(args ...any) *slog.Logger
}

var (
	app    *fx.App
	h      host.Host
	logger log.Logger
	dialer client.Dialer
)

var flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "addr",
		Aliases: []string{"a"},
		Usage:   "static bootstrap `ADDR`",
		EnvVars: []string{"WW_ADDR"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap discovery `ADDR`",
		Value:   bootstrapAddr(),
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.DurationFlag{
		Name:    "timeout",
		Usage:   "dial timeout",
		Value:   time.Second * 15,
		EnvVars: []string{"WW_CLIENT_TIMEOUT"},
	},
}

func Command(log Logger) *cli.Command {
	return &cli.Command{
		Name:    "cluster",
		Usage:   "cli client for wetware clusters",
		Aliases: []string{"client"}, // TODO(soon):  deprecate
		Flags:   flags,
		Subcommands: []*cli.Command{
			run(log),
		},
	}
}

func setup(log Logger) cli.BeforeFunc {
	return func(c *cli.Context) (err error) {

		ctx, cancel := context.WithTimeout(
			c.Context,
			c.Duration("timeout"))
		defer cancel()

		ch, err := clientHost(c)
		if err != nil {
			return err
		}
		defer ch.Close()

		conn, err := dial(c, log, ch)
		if err != nil {
			return err
		}
		defer conn.Close()

		h = host.Host(conn.Bootstrap(c.Context))
		defer h.Release()

		return app.Start(ctx)
	}
}

func teardown() cli.AfterFunc {
	return func(c *cli.Context) (err error) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			app.StopTimeout())
		defer cancel()

		h.Release()

		return app.Stop(ctx)
	}
}

func dial(c *cli.Context, log Logger, h local.Host) (*rpc.Conn, error) {
	return client.Dialer{
		NS:       c.String("ns"),
		Peers:    c.StringSlice("peer"),
		Discover: c.String("discover"),
		Logger: log.With(
			"peers", c.StringSlice("peer"),
			"discover", c.String("discover")),
	}.Dial(c.Context, h)
}

func clientHost(c *cli.Context) (local.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport))
}

func bootstrapAddr() string {
	return path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())
}

func loopback() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		return "lo"
	}
}
