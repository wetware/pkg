package start

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	log "github.com/lthibault/log/pkg"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	ctxutil "github.com/lthibault/wetware/internal/util/ctx"
	logutil "github.com/lthibault/wetware/internal/util/log"
	"github.com/lthibault/wetware/pkg/server"
)

var (
	proc = ctxutil.WithLifetime(context.Background())
	host server.Host
)

// Init the `start` command
func Init() cli.BeforeFunc {
	return func(c *cli.Context) error {
		log := logutil.New(c)

		host = server.New(
			server.WithLogger(log),
			withTracer(log),
		)

		return nil
	}
}

// Flags for the `start` command
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "repo",
			Aliases: []string{"r"},
			Usage:   "path to IPFS repository",
			EnvVars: []string{"WW_REPO"},
		},
	}
}

// Run the `start` command
func Run() cli.ActionFunc {
	return func(c *cli.Context) error {
		if err := host.Start(); err != nil {
			return errors.Wrap(err, "start host")
		}

		host.Log().Info("host started")
		<-proc.Done()
		host.Log().Warn("host shutting down")

		if err := host.Close(); err != nil {
			return errors.Wrap(err, "stop host")
		}

		return nil
	}
}

func withTracer(log log.Logger) server.Option {
	ev := []interface{}{
		new(event.EvtLocalAddressesUpdated),
		new(event.EvtPeerIdentificationCompleted),
		new(event.EvtPeerIdentificationFailed),
	}

	return server.WithEventHandler(ev, func(v interface{}) {
		switch ev := v.(type) {
		case event.EvtLocalAddressesUpdated:
			as := make([]multiaddr.Multiaddr, len(ev.Current))
			for i, a := range ev.Current {
				as[i] = a.Address
			}

			log.
				WithField("addrs", as).
				Info("host listening")
		case event.EvtPeerIdentificationCompleted:
			log.
				WithField("peer", ev.Peer).
				Info("identification succeeded")
		case event.EvtPeerIdentificationFailed:
			log.
				WithError(ev.Reason).
				WithField("peer", ev.Peer).
				Warn("identification failed")
		}
	})
}
