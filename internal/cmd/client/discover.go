package client

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/lthibault/log"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
)

// ww client discover scan -s tcp://127.0.0.0:8822/24
func Scan() *cli.Command {
	var (
		h host.Host
		d discovery.Discoverer
	)

	return &cli.Command{
		Name:   "scan",
		Usage:  "scan an IP range for cluster hosts",
		Flags:  scanFlags,
		Before: beforeScan(&d, &h),
		Action: scan(&d),
		After:  afterScan(&h),
	}
}

var scanFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
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
		Value:   "/ip4/228.8.8.8/udp/8822/survey", // TODO:  this should default to survey
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.DurationFlag{
		Name:  "timeout",
		Usage: "per-connection timeout",
		Value: time.Millisecond * 10,
	},
}

// ww client discover publish --auto
func Publish() *cli.Command {
	return &cli.Command{
		Name:   "publish",
		Usage:  "publish a peer record",
		Flags:  publishFlags,
		Action: publish,
	}
}

var publishFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "net",
		Value: "tcp4",
	},
	&cli.StringFlag{
		Name:  "host",
		Value: "0.0.0.0",
	},
	&cli.IntFlag{
		Name:  "port",
		Value: 8822,
	},
	&cli.BoolFlag{
		Name:  "auto",
		Usage: "autogenerate a peer record for testing",
	},
}

/*
	SCAN
*/

func beforeScan(d *discovery.Discoverer, h *host.Host) cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		maddr, err := multiaddr.NewMultiaddr(c.String("discover"))
		if err != nil {
			return err
		}

		*h, err = libp2p.New(c.Context,
			libp2p.NoTransports,
			libp2p.Transport(libp2pquic.NewTransport),
			libp2p.ListenAddrStrings(c.StringSlice("listen")...))
		if err != nil {
			return err
		}

		*d, err = boot.Parse(*h, maddr)
		return err
	}
}

func scan(d *discovery.Discoverer) cli.ActionFunc {
	return func(c *cli.Context) error {
		peers, err := (*d).FindPeers(c.Context, c.String("ns"))
		if err != nil {
			return err
		}

		enc := json.NewEncoder(c.App.Writer)
		for info := range peers {
			if err := enc.Encode(info); err != nil {
				return err
			}
		}

		return nil
	}
}

func afterScan(h *host.Host) cli.AfterFunc {
	return func(c *cli.Context) error {
		if *h != nil {
			return (*h).Close()
		}
		return nil
	}
}

/*
	PUBLISH
*/

type recordPublisher struct {
	payload []byte
}

func publish(c *cli.Context) error {
	var p recordPublisher
	if c.Bool("auto") {
		if err := p.autoGenerate(); err != nil {
			return err
		}
	} else {
		return errors.New("TODO: add support for reading signed envelopes from stdin")
	}

	netloc := fmt.Sprintf("%s:%d",
		c.String("host"),
		c.Int("port"))

	l, err := new(net.ListenConfig).Listen(c.Context, c.String("net"), netloc)
	if err != nil {
		return err
	}

	go func() {
		defer l.Close()
		<-c.Done()
	}()

	logger.WithField("addr", l.Addr()).Info("serving")

	for {
		conn, err := l.Accept()
		if err != nil {
			if c.Context.Err() != nil {
				break
			}

			return err
		}

		go p.handle(conn)
	}

	return nil
}

func (p *recordPublisher) handle(conn net.Conn) {
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(time.Millisecond * 10)); err != nil {
		logger.WithError(err).Debug("error serving conn")
	}

	if _, err := io.Copy(conn, bytes.NewReader(p.payload)); err != nil {
		logger.WithError(err).Debug("error writing payload")
	}
}

func (p *recordPublisher) autoGenerate() error {
	pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return err
	}

	id, err := peer.IDFromPrivateKey(pk)
	if err != nil {
		return err
	}

	var rec = peer.PeerRecord{
		PeerID: id,
		Seq:    uint64(time.Now().UnixNano()),
		Addrs: []multiaddr.Multiaddr{
			multiaddr.StringCast(fmt.Sprintf("/ip4/127.0.0.1/udp/2020/p2p/%s", id)),
		},
	}

	logger.With(log.F{
		"seq": rec.Seq,
		"id":  id,
	}).Info("generated record")

	e, err := record.Seal(&rec, pk)
	if err != nil {
		return err
	}

	p.payload, err = e.Marshal()
	return err
}
