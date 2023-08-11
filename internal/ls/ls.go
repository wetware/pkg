package ls

import (
	"fmt"

	"capnproto.org/go/capnp/v3"
	"golang.org/x/exp/slog"

	"github.com/urfave/cli/v2"

	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/wetware/pkg/cap/auth"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
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

func Command(log Logger) *cli.Command {
	return &cli.Command{
		Name: "ls",
		Action: func(c *cli.Context) error {
			host, err := authenticate(c, log)
			if err != nil {
				return err
			}
			defer host.Release()

			// the view will be a null client if authentication succeeded,
			// but you don't have permission to access the capability.
			// Authentication errors are reported as RPC exceptions.
			view, release := host.View(c.Context)
			defer release()

			// TODO(performance):  remove this block once we
			// are confident that promise pipelining works.
			if err := capnp.Client(view).Resolve(c.Context); err != nil {
				return fmt.Errorf("resolve: %w", err)
			}

			// optimistically iterate through the view.  Under the hood
			// this uses Cap'n Proto streaming RPC semantics, along with
			// BBR flow control.
			it, release := view.Iter(c.Context, query(c))
			defer release()

			// render a view of the iterator.
			return render(c, it)
		},
	}
}

func authenticate(c *cli.Context, log Logger) (host.Host, error) {
	h, err := clientHost(c)
	if err != nil {
		return failure(err)
	}

	conn, err := client.Dialer{
		NS:       c.String("ns"),
		Peers:    c.StringSlice("peer"),
		Discover: c.String("discover"),
		Logger: log.With(
			"peers", c.StringSlice("peer"),
			"discover", c.String("discover")),
	}.Dial(c.Context, h)
	if err != nil {
		defer h.Close()
		return failure(err)
	}

	go func() {
		defer h.Close()
		defer conn.Close()

		select {
		case <-c.Done():
		case <-conn.Done():
		}
	}()

	client := conn.Bootstrap(c.Context)

	// TODO(performance):  remove when we've addressed all issues with
	// promise pipelining
	if err := client.Resolve(c.Context); err != nil {
		return failure(err)
	}

	// XXX:  pass in the appropriate signer
	sess, release := auth.Provider(client).Provide(c.Context, auth.Signer{})
	defer release()

	view := sess.View()
	err = capnp.Client(view).Resolve(c.Context) // DEBUG
	return sess.Host(), err
}

func failure(err error) (host.Host, error) {
	return host.Host(capnp.ErrorClient(err)), err
}

func clientHost(c *cli.Context) (local.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport))
}

func signer(c *cli.Context) auth.Signer {
	return auth.Signer{}
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, it view.Iterator) error {
	// range over the iterator; this will block when waiting
	// for data from the network.
	//
	// TODO(performance):  ensure we're doing some kind of sensible batching
	for r := it.Next(); r != nil; r = it.Next() {
		fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
	}

	return it.Err()
}
