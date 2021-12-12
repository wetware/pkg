package boot

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"runtime"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	"github.com/lthibault/log"
)

var token = make(chan struct{}, 8)

func init() {
	for len(token) < cap(token) {
		token <- struct{}{}
	}
}

// Type RoundTripper can test a port by sending a UDP packet.
type RoundTripper struct {
	Port    int
	Timeout time.Duration
	Request Knock
}

func (rt RoundTripper) RoundTrip(ctx context.Context, conn *net.UDPConn, host *net.UDPAddr, b []byte) (n int, err error) {
	if err = conn.SetWriteDeadline(rt.deadline(ctx)); err != nil {
		return
	}

	n, err = conn.WriteToUDP(rt.Request.Bytes(), host)
	if err != nil {
		err = fmt.Errorf("send: %w", err)
		return
	}

	if conn.SetReadDeadline(rt.deadline(ctx)); err != nil {
		return
	}

	if n, _, err = conn.ReadFromUDP(b); err != nil {
		err = fmt.Errorf("recv: %w", err)
		return
	}

	return
}

func (p RoundTripper) deadline(ctx context.Context) (t time.Time) {
	var ok bool
	if t, ok = ctx.Deadline(); ok {
		return
	}

	if p.Timeout <= 0 {
		p.Timeout = time.Millisecond * 500
	}

	return time.Now().Add(p.Timeout)
}

// Scanner is a discovery strategy that attempts to dial
// a specific port for each IP in a given CIDR range, by
// default: 10.0.0.0/24.
//
// To avoid overwhelming hosts, the CIDR range is traversed
// in pseudorandom order, with bounded concurrency.
type Scanner struct {
	Logger log.Logger
	Port   int
	CIDR   string
}

func (s Scanner) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"port": s.Port,
		"cidr": s.CIDR,
	}
}

func (s Scanner) FindPeers(ctx context.Context, ns string, opts ...discovery.Option) (<-chan peer.AddrInfo, error) {
	o := discovery.Options{}
	if err := o.Apply(opts...); err != nil {
		return nil, err
	}

	_, ipnet, err := net.ParseCIDR(s.CIDR)
	if err != nil {
		return nil, err
	}

	out := make(chan peer.AddrInfo, 1)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(out)

		var (
			// Convert IPNet struct mask and address to uint32.
			// Network is BigEndian.
			mask  = binary.BigEndian.Uint32(ipnet.Mask)
			begin = binary.BigEndian.Uint32(ipnet.IP)
			end   = (begin & mask) | (mask ^ 0xffffffff) // final address

			// Each IP will be masked with the nonce before knocking.
			// This effectively randomizes the search.
			nonce = rand.Uint32() & (mask ^ 0xffffffff)
		)

		// loop through CIDR as unt32
		for i := begin; i <= end; i++ {
			select {
			case <-token:
				// Skip X.X.X.0 and X.X.X.255
				if i^nonce == begin || i^nonce == end {
					continue
				}

				go s.roundTrip(ctx, ns, i^nonce, out, cancel)

			case <-ctx.Done():
				s.Logger.WithError(err).Trace("interrupt")
				return
			}
		}
	}()

	return out, nil
}

func (s Scanner) roundTrip(ctx context.Context, ns string, i uint32, out chan<- peer.AddrInfo, abort context.CancelFunc) {
	defer func() {
		select {
		case token <- struct{}{}:
		default:
		}
	}()

	var ip = make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, i)

	s.Logger.Tracef("knocking on %s:%d", ip, s.Port)

	var (
		peer peer.PeerRecord
	)

	if err := s.RoundTrip(ctx, ns, ip, &peer); err != nil {
		return
	}

	if consume(ctx, out, &peer) {
		abort()
	}
}

// RoundTrip sends 'k' to the 's.Port' on host 'addr' and waits for a
// reply until 'ctx' expires.
func (s Scanner) RoundTrip(ctx context.Context, ns string, ip net.IP, r record.Record) error {
	request, err := NewKnock(ns)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	var knocker = RoundTripper{
		Port:    s.Port,
		Request: request,
	}

	conn, err := s.connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var (
		b  [bufsize]byte
		n  int
		ne net.Error
	)

	// Round-trip and swallow errors due to socket timeouts.
	if n, err = knocker.RoundTrip(ctx, conn, &net.UDPAddr{
		Port: s.Port,
		IP:   ip,
	}, b[:]); errors.As(err, &ne) && ne.Timeout() {
		return context.DeadlineExceeded
	} else if err != nil {
		return err
	}

	// FIXME(security):  boot packets should have their own Record type.
	//
	// Risk:  Low
	//
	// A possible attack scenario involves returning records from other peers.
	// This could be used to place a disproportionate load on specific peers,
	// thus opening up a vector for targeted DDoS.
	//
	// This attack is only possible if attackers can obtain valid a valid, peer
	// record for the target node.
	if _, err = record.ConsumeTypedEnvelope(b[:n], r); err != nil {
		return fmt.Errorf("consume envelope: %w", err)
	}

	return nil
}

func consume(ctx context.Context, out chan<- peer.AddrInfo, r *peer.PeerRecord) bool {
	select {
	case out <- peer.AddrInfo{ID: r.PeerID, Addrs: r.Addrs}:
		return true

	case <-ctx.Done():
		return false
	}
}

func (s Scanner) connection(ctx context.Context) (*net.UDPConn, error) {
	// Listen on all non-multicast IPs.
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}

	_ = conn.SetReadBuffer(bufsize)
	_ = conn.SetWriteBuffer(bufsize)

	// Ensure the connection is closed if it is released early.
	runtime.SetFinalizer(conn, func(c io.Closer) { c.Close() })

	return conn, err
}
