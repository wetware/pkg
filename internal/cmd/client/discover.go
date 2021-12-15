package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	"github.com/urfave/cli/v2"
	logutil "github.com/wetware/ww/internal/util/log"
	"github.com/wetware/ww/pkg/boot"
)

// ww client discover scan -p 8822
func Scan() *cli.Command {
	var s boot.Context

	return &cli.Command{
		Name:   "scan",
		Usage:  "scan a port range for cluster hosts",
		Flags:  scanFlags,
		Before: beforeScan(&s),
		Action: scan(&s),
	}
}

var scanFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "namespace to query",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.StringFlag{
		Name:    "cidr",
		Usage:   "CIDR range to scan",
		Value:   "127.0.0.0/24",
		EnvVars: []string{"WW_DISCOVER_CIDR"},
	},
	&cli.IntFlag{
		Name:    "port",
		Usage:   "port to scan",
		Value:   8822,
		EnvVars: []string{"WW_DISCOVER_PORT"},
	},
}

var secret crypto.PrivKey

func init() {
	var err error
	if secret, _, err = crypto.GenerateECDSAKeyPair(rand.Reader); err != nil {
		panic(err)
	}
}

type recordScanner struct{}

func (recordScanner) Handle(conn net.Conn, src record.Record) (*record.Envelope, error) {
	e, err := record.Seal(src, secret)
	if err != nil {
		return e, err
	}

	data, err := e.Marshal()
	if err != nil {
		return e, err
	}

	_, err = io.Copy(conn, bytes.NewReader(data))
	return e, err
}

func (recordScanner) Scan(conn net.Conn, dst record.Record) (*record.Envelope, error) {
	data, err := ioutil.ReadAll(io.LimitReader(conn, 512)) // arbitrary MTU
	if err != nil {
		return nil, err
	}

	return record.ConsumeTypedEnvelope(data, dst)
}

func beforeScan(s *boot.Context) cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		s.Strategy = &boot.ScanSubnet{
			CIDR:    boot.CIDR{Subnet: c.String("cidr")},
			Port:    c.Int("port"),
			Handler: recordScanner{},
		}

		return
	}
}

func scan(b *boot.Context) cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			timeout = c.Duration("timeout")
			cancel  context.CancelFunc
		)
		c.Context, cancel = context.WithTimeout(c.Context, timeout)
		defer cancel()

		c.Context, cancel = signal.NotifyContext(c.Context,
			syscall.SIGINT,
			syscall.SIGTERM)
		defer cancel()

		rs := make(chan peer.PeerRecord, 1)
		defer close(rs)

		go func() {
			var (
				rec    peer.PeerRecord
				logger = logutil.New(c)
			)

			for {

				_, err := b.Strategy.Scan(c.Context, new(net.Dialer), &rec)
				if err != nil {
					logger.Info(err)
					continue
				}

				select {
				case rs <- rec:
				case <-c.Context.Done():
					return
				}
			}

		}()

		enc := json.NewEncoder(c.App.Writer)
		enc.SetIndent("\n", "  ")

		for record := range rs {
			if err := enc.Encode(record); err != nil {
				return err
			}
		}

		return nil

		// s.RequestBody, err = record.Seal(&rec, pk)
		// if err != nil {
		// 	return err
		// }

		// ps, err := s.FindPeers(c.Context, c.String("ns"))
		// if err != nil {
		// 	return err
		// }

		// enc := json.NewEncoder(c.App.Writer)
		// enc.SetIndent("\n", "  ")

		// for peer := range ps {
		// 	if err := enc.Encode(peer); err != nil {
		// 		return err
		// 	}
		// }

		// if errors.Is(c.Context.Err(), context.Canceled) {
		// 	return nil
		// }

		// return c.Context.Err()
	}
}
