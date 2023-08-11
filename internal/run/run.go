package run

import (
	"errors"
	"os"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/cap/auth"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/rom"
	"golang.org/x/exp/slog"
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

var flags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "dial",
		Usage:   "connect to cluster",
		EnvVars: []string{"WW_DIAL"},
	},
	&cli.BoolFlag{
		Name:     "stdin",
		Aliases:  []string{"s"},
		Usage:    "load system image from stdin",
		Category: "ROM",
	},
}

func Command(log Logger) *cli.Command {
	return &cli.Command{
		Name:   "run",
		Usage:  "execute a local webassembly process",
		Flags:  flags,
		Action: run(log),
	}
}

func run(log Logger) cli.ActionFunc {
	return func(c *cli.Context) error {
		client, err := dial(c, log)
		if err != nil {
			return err
		}
		defer client.Release()

		wetware := ww.Ww[host.Host]{
			Log:    log,
			NS:     c.String("ns"),
			Stdin:  c.App.Reader,
			Stdout: c.App.Writer,
			Stderr: c.App.ErrWriter,
			Client: client,
		}

		rom, err := bytecode(c)
		if err != nil {
			return err
		}

		// run without connecting to a cluster
		return wetware.Exec(c.Context, rom)
	}
}

func dial(c *cli.Context, log Logger) (host.Host, error) {
	// dial into a cluster?
	if c.Bool("dial") {
		return authenticate(c, log)
	}

	// we're not connected to the cluster;  return an
	// auth.Provider that immediately fails with a helpful
	// message.
	return failure(errors.New("disconnected"))
}

func failure(err error) (host.Host, error) {
	return host.Host(capnp.ErrorClient(err)), err
}

// authenticate implements the client-side auth flow.
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
		return failure(err)
	}

	go func() {
		defer h.Close()
		defer conn.Close()

		select {
		case <-conn.Done():
		case <-c.Context.Done():
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

	return sess.Host(), nil
}

func clientHost(c *cli.Context) (local.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport))
}

func bytecode(c *cli.Context) (ww.ROM, error) {
	if c.Bool("stdin") {
		return ww.Read(c.App.Reader)
	}

	// file?
	if c.Args().Len() > 0 {
		return loadROM(c)
	}

	// use the default bytecode
	return rom.Default(), nil
}

func loadROM(c *cli.Context) (ww.ROM, error) {
	f, err := os.Open(c.Args().First())
	if err != nil {
		return ww.ROM{}, err
	}
	defer f.Close()

	return ww.Read(f)
}
