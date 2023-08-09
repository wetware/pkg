package socket

import (
	"context"
	"net"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	"golang.org/x/time/rate"
)

type RecordValidator func(*record.Envelope, *Record) error

func BasicValidator(self peer.ID) RecordValidator {
	return func(e *record.Envelope, r *Record) error {
		peer, err := r.Peer()
		if err != nil {
			return err
		}

		if self != "" && peer == self {
			return ErrIgnore
		}

		// envelope was signed by peer?
		if !peer.MatchesPublicKey(e.PublicKey) {
			err = record.ErrInvalidSignature
		}

		return err
	}
}

// RateLimiter provides flow-control for a Socket.
type RateLimiter struct {
	lim    *rate.Limiter
	tokens func(int) int // allows us to e.g. convert bytes -> packets
}

func NewRateLimiter(r rate.Limit, burst int, f func(int) int) *RateLimiter {
	return &RateLimiter{
		lim:    rate.NewLimiter(r, burst),
		tokens: f,
	}
}

// NewBandwidthLimiter enforces limits over the network bandwidth used
// by a socket.
//
// NOTE:  limit and burst are expressed in *bits* per second and *bits*,
//        respectively.  Do not confuse this with bytes.
func NewBandwidthLimiter(r rate.Limit, burst int) *RateLimiter {
	return NewRateLimiter(r, burst, func(n int) int { return n * 8 })
}

// NewPacketLimiter enforces limits over the number of packets sent
// by a socket.  Units are packets/sec and packets, respectively.
func NewPacketLimiter(r rate.Limit, burst int) *RateLimiter {
	return NewRateLimiter(r, burst, func(n int) int { return 1 })
}

// Reserve a slot for an outgoing message of size n bytes.
func (r *RateLimiter) Reserve(ctx context.Context, n int) (err error) {
	if r != nil {
		err = r.lim.WaitN(ctx, r.tokens(n))
	}

	return
}

type recordConn struct {
	lim      *RateLimiter
	validate RecordValidator
	net.PacketConn
}

// Send writes the message m to addr.  Send does not support write
// timeouts since PacketConn implementations are expected to flush
// their write-buffers quickly. The context is used to abort while
// waiting for the rate-limiter to allocate resources.
//
// Implementations MUST support concurrent calls to Send.
func (conn recordConn) Send(ctx context.Context, e *record.Envelope, addr net.Addr) error {
	b, err := e.Marshal()
	if err != nil {
		return err
	}

	if err = conn.lim.Reserve(ctx, len(b)); err != nil {
		return err
	}

	_, err = conn.WriteTo(b, addr)
	return err
}

// Scan a record from the from the Socket, returning the originator's
// along with the signed envelope.
//
// Callers MUST NOT make concurrent calls to Recv.
func (conn recordConn) Scan(p *Record) (net.Addr, error) {
	var buf [maxDatagramSize]byte
	n, addr, err := conn.ReadFrom(buf[:])
	if err != nil {
		return nil, err
	}

	e, err := record.ConsumeTypedEnvelope(buf[:n], p)
	if err != nil {
		return nil, ValidationError{
			Cause: err,
			From:  addr,
		}
	}

	if err = conn.validate(e, p); err != nil {
		return nil, ValidationError{
			Cause: err,
			From:  addr,
		}
	}

	return addr, nil
}
