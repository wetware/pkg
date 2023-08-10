package run

import (
	"errors"
	"os"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/pkg"
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
		Name:    "join",
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
		wetware := ww.Ww{
			Log:    log,
			NS:     c.String("ns"),
			Stdin:  c.App.Reader,
			Stdout: c.App.Writer,
			Stderr: c.App.ErrWriter,
		}

		rom, err := bytecode(c)
		if err != nil {
			return err
		}

		// dial into a cluster?
		if c.Bool("dial") {
			return dialAndExec(c, log, wetware, rom)
		}

		wetware.Client = capnp.ErrorClient(errors.New("NOT IMPLEMENTED"))
		// run without connecting to a cluster
		return wetware.Exec(c.Context, rom)
	}
}

func dialAndExec(c *cli.Context, log Logger, wetware ww.Ww, rom ww.ROM) error {
	h, err := clientHost(c)
	if err != nil {
		return err
	}
	defer h.Close()

	conn, err := client.Dialer{
		NS:       c.String("ns"),
		Peers:    c.StringSlice("peer"),
		Discover: c.String("discover"),
		Logger: log.With(
			"peers", c.StringSlice("peer"),
			"discover", c.String("discover")),
	}.Dial(c.Context, h)
	if err != nil {
		return err
	}
	defer conn.Close()

	wetware.Client = conn.Bootstrap(c.Context)
	return wetware.Exec(c.Context, rom)
}

func clientHost(c *cli.Context) (host.Host, error) {
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
