package client

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/boot"
)

// ww client discover scan -p 8822
func Scan() *cli.Command {
	var k boot.PortKnocker

	return &cli.Command{
		Name:   "scan",
		Usage:  "scan a port range for cluster hosts",
		Flags:  scanFlags,
		Before: beforeScan(&k),
		Action: scan(&k),
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

func beforeScan(k *boot.PortKnocker) cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		k.Port = c.Int("port")
		k.Logger = logger.
			WithField("ns", c.String("ns")).
			WithField("port", c.Int("port")).
			WithField("cidr", c.String("cidr"))
		k.Logger.Debug("port scan started")
		_, k.Subnet, err = net.ParseCIDR(c.String("cidr"))
		return
	}
}

func scan(k *boot.PortKnocker) cli.ActionFunc {
	return func(c *cli.Context) error {
		// HACK: send empty peer record signed with ephemeral key as knock payload.
		// At some point we should probably query a specific namespace, which will
		// involve its own record type.
		var rec peer.PeerRecord
		pk, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
		if err != nil {
			return err
		}
		k.RequestBody, err = record.Seal(&rec, pk)
		if err != nil {
			return err
		}

		ps, err := k.FindPeers(c.Context, c.String("ns"))
		if err != nil {
			return err
		}

		enc := json.NewEncoder(c.App.Writer)
		enc.SetIndent("\n", "  ")

		for peer := range ps {
			if err := enc.Encode(peer); err != nil {
				return err
			}
		}

		if errors.Is(c.Context.Err(), context.Canceled) {
			return nil
		}

		return c.Context.Err()
	}
}
