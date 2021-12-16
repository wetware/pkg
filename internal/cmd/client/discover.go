package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	"github.com/lthibault/log"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
)

// ww client discover scan -s tcp://127.0.0.0:8822/24
func Crawl() *cli.Command {
	var b boot.Crawler

	return &cli.Command{
		Name:   "crawl",
		Usage:  "scan an IP range for cluster hosts",
		Flags:  scanFlags,
		Before: beforeScan(&b),
		Action: scan(&b),
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
		Name:  "subnet",
		Usage: "CIDR range to scan",
		Value: "127.0.0.0/24",
		// Aliases: []string{"-s"},
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringFlag{
		Name:  "net",
		Value: "tcp4",
	},
	&cli.IntFlag{
		Name:  "port",
		Usage: "port to scan",
		Value: 8822,
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

type recordScanner struct{}

func (recordScanner) Scan(conn net.Conn, dst record.Record) (*record.Envelope, error) {
	data, err := ioutil.ReadAll(io.LimitReader(conn, 512)) // arbitrary MTU
	if err != nil {
		return nil, err
	}

	return record.ConsumeTypedEnvelope(data, dst)
}

func beforeScan(s *boot.Crawler) cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		s.Strategy = &boot.ScanSubnet{
			Net:     c.String("net"),
			Port:    c.Int("port"),
			Subnet:  boot.Subnet{CIDR: c.String("subnet")},
			Scanner: recordScanner{},
		}

		return
	}
}

func scan(b *boot.Crawler) cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			rs    = make(chan peer.PeerRecord, 1)
			cherr = make(chan error, 1)
		)

		go func() {
			defer close(rs)
			defer close(cherr)

			var (
				rec    peer.PeerRecord
				dialer = logDialer{Logger: logger}
			)

			_, err := b.Strategy.Scan(c.Context, dialer, &rec)
			if err != nil {
				cherr <- err // buffered
			}
		}()

		enc := json.NewEncoder(c.App.Writer)
		enc.SetIndent("\n", "  ")

		for {
			select {
			case record := <-rs:
				if err := enc.Encode(record); err != nil {
					return err
				}

			case err := <-cherr:
				return err
			}
		}
	}
}

type logDialer struct {
	Logger log.Logger
}

func (d logDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	log := d.Logger.With(log.F{
		"net":  network,
		"addr": addr,
	})
	log.Trace("dialing")

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, network, addr)
	if err == nil {
		return conn, nil
	}

	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		log.Debug("no answer")
		return nil, boot.ErrSkip
	}

	return nil, err
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
	pk, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		return err
	}

	id, err := peer.IDFromPrivateKey(pk)
	if err != nil {
		return err
	}

	var rec = peer.PeerRecord{
		Seq: uint64(time.Now().UnixNano()),
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
