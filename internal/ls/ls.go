package ls

import (
	"fmt"

	"capnproto.org/go/capnp/v3/rpc"
	"golang.org/x/exp/slog"

	"github.com/urfave/cli/v2"

	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/cluster/routing"
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

func Command(log Logger) *cli.Command {
	return &cli.Command{
		Name: "ls",
		Action: func(c *cli.Context) error {
			h, err := clientHost(c)
			if err != nil {
				return err
			}
			defer h.Close()

			conn, err := dial(c, h)
			if err != nil {
				return err
			}
			defer conn.Close()

			peer := host.Host(conn.Bootstrap(c.Context))
			defer peer.Release()

			view, release := peer.View(c.Context)
			defer release()

			it, release := view.Iter(c.Context, query(c))
			defer release()

			for r := it.Next(); r != nil; r = it.Next() {
				render(c, r)
			}

			return it.Err()
		},
	}
}

func dial(c *cli.Context, h local.Host) (*rpc.Conn, error) {
	return client.Dialer{
		Logger:   slog.Default(),
		NS:       c.String("ns"),
		Peers:    c.StringSlice("peer"),
		Discover: c.String("discover"),
	}.Dial(c.Context, h)
}

func clientHost(c *cli.Context) (local.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport))
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, r routing.Record) {
	fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
}
