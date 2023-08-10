package crawl

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/record"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/wetware/pkg/boot/socket"
)

const P_CIDR = 103

func init() {
	if err := ma.AddProtocol(ma.Protocol{
		Name:       "cidr",
		Code:       P_CIDR,
		VCode:      ma.CodeToVarint(P_CIDR),
		Size:       8, // bits
		Transcoder: TranscoderCIDR{},
	}); err != nil {
		panic(err)
	}
}

var (
	// ErrCIDROverflow is returned when a CIDR block is too large.
	ErrCIDROverflow = errors.New("CIDR overflow")
)

type Crawler struct {
	host host.Host
	sock *socket.Socket
	iter Strategy
}

func New(h host.Host, conn net.PacketConn, s Strategy, opt ...socket.Option) *Crawler {
	c := &Crawler{
		host: h,
		iter: s,
		sock: socket.New(conn, withDefault(h, opt)...),
	}

	c.sock.Bind(c.handler())

	return c
}

func withDefault(h host.Host, opt []socket.Option) []socket.Option {
	return append([]socket.Option{
		socket.WithRateLimiter(socket.NewPacketLimiter(32, 8)),
		socket.WithValidator(socket.BasicValidator(h.ID())),
	}, opt...)
}

func (c *Crawler) Close() error {
	return c.sock.Close()
}

func (c *Crawler) handler() socket.RequestHandler {
	return func(r socket.Request) error {
		return c.sock.SendResponse(c.sealer, c.host, r.From, r.NS)
	}
}

func (c *Crawler) Advertise(_ context.Context, ns string, opt ...discovery.Option) (ttl time.Duration, err error) {
	if len(c.host.Addrs()) == 0 {
		return 0, errors.New("no listen addrs")
	}

	var opts = discovery.Options{Ttl: peerstore.TempAddrTTL}
	if err = opts.Apply(opt...); err != nil {
		return
	}

	if err = c.sock.Track(ns, opts.Ttl); err == nil {
		ttl = opts.Ttl
	}

	return
}

func (c *Crawler) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	var opts discovery.Options
	if err := opts.Apply(opt...); err != nil {
		return nil, err
	}

	iter, err := c.iter()
	if err != nil {
		return nil, err
	}

	out, cancel := c.sock.Subscribe(ns, opts.Limit)
	go func() {
		defer cancel()

		var (
			addr net.UDPAddr
			id   = c.host.ID()
		)

		// Iterate through the IP range and send request packets.
		// This is rate-limited by the socket.
		for c.active(ctx) && iter.Next(&addr) {
			switch err := c.sock.SendRequest(ctx, c.sealer, &addr, id, ns); err {
			case nil:
				// Packet sent.  Keep crawling.
				c.sock.Log().
					WithField("to", &addr).
					Trace("sent request packet")
				continue

			case context.Canceled:
				// Graceful abort.  The caller cancels the context when it
				// has found enough peers.
				c.sock.Log().
					Trace("peer discovery finished")

			case context.DeadlineExceeded:
				// Timeout.  The caller hasn't found enough peers, but has
				// timed out.  This isn't always an error.  In most cases,
				// the caller will know what to do.
				c.sock.Log().
					WithField("reason", err).
					Debug("peer discovery aborted")

			default:
				// Any other error indicates a failure to send the request
				// packet.  Something definitely went wrong.
				c.sock.Log().
					WithError(err).
					WithField("to", &addr).
					Error("failed to send request packet")
			}

			return
		}

		// Wait for response
		select {
		case <-ctx.Done():
		case <-c.sock.Done():
		}
	}()

	return out, nil
}

func (c *Crawler) active(ctx context.Context) (ok bool) {
	select {
	case <-ctx.Done():
	case <-c.sock.Done():
	default:
		ok = true
	}

	return
}

func (c *Crawler) sealer(r record.Record) (*record.Envelope, error) {
	return record.Seal(r, privkey(c.host))
}

func privkey(h host.Host) crypto.PrivKey {
	return h.Peerstore().PrivKey(h.ID())
}

// TranscoderCIDR decodes a uint8 CIDR block
type TranscoderCIDR struct{}

func (ct TranscoderCIDR) StringToBytes(cidrBlock string) ([]byte, error) {
	num, err := strconv.ParseUint(cidrBlock, 10, 8)
	if err != nil {
		return nil, err
	}

	if num > 128 {
		return nil, ErrCIDROverflow
	}

	return []byte{uint8(num)}, err
}

func (ct TranscoderCIDR) BytesToString(b []byte) (string, error) {
	if len(b) > 1 || b[0] > 128 {
		return "", ErrCIDROverflow
	}

	return strconv.FormatUint(uint64(b[0]), 10), nil
}

func (ct TranscoderCIDR) ValidateBytes(b []byte) error {
	if uint8(b[0]) > 128 { // 128 is maximum CIDR block for IPv6
		return ErrCIDROverflow
	}

	return nil
}
