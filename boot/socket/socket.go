// Package socket implements signed sockets for bootstrap services.
package socket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	ctxutil "github.com/lthibault/util/ctx"
)

func init() { close(closedChan) }

const maxDatagramSize = 2 << 10 // KB

type RequestHandler func(Request) error

// Socket is a a packet-oriented network interface that exchanges
// signed messages.
type Socket struct {
	log  Logger
	done chan struct{}
	conn recordConn

	tick  *time.Ticker
	cache *RecordCache

	handleError func(*Socket, error)

	mu   sync.RWMutex
	subs map[string]subscriberSet
	advt map[string]time.Time
	time time.Time
}

// New socket.  The wrapped PacketConn implementation MUST flush
// its send buffer in a timely manner.  It must also provide
// unreliable delivery semantics; if the underlying transport is
// reliable, it MUST suppress any errors due to failed connections
// or delivery.  The standard net.PacketConn implementations satisfy
// these condiitions
func New(conn net.PacketConn, opt ...Option) *Socket {
	sock := &Socket{
		conn: recordConn{PacketConn: conn},
		time: time.Now(),
		tick: time.NewTicker(time.Millisecond * 500),
		done: make(chan struct{}),
		advt: make(map[string]time.Time),
		subs: make(map[string]subscriberSet),
	}

	for _, option := range withDefault(opt) {
		option(sock)
	}

	return sock
}

// Bind the handler to the socket and begin servicing incoming
// requests.  Bind MUST NOT be called more than once.
func (s *Socket) Bind(h RequestHandler) {
	go s.tickloop()
	go s.serve(h)
}

func (s *Socket) Done() <-chan struct{} {
	return s.done
}

func (s *Socket) Log() Logger { return s.log }

func (s *Socket) Close() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
	default:
		s.tick.Stop()
		close(s.done)
		err = s.conn.Close()
	}

	return
}

func (s *Socket) Track(ns string, ttl time.Duration) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		err = fmt.Errorf("already %s", ErrClosed)

	default:
		s.advt[ns] = s.time.Add(ttl)
	}

	return
}

func (s *Socket) Send(ctx context.Context, e *record.Envelope, addr net.Addr) error {
	return s.conn.Send(ctx, e, addr)
}

func (s *Socket) SendRequest(ctx context.Context, seal Sealer, addr net.Addr, id peer.ID, ns string) error {
	e, err := s.cache.LoadRequest(seal, id, ns)
	if err != nil {
		return err
	}

	return s.Send(ctx, e, addr)
}

func (s *Socket) SendSurveyRequest(ctx context.Context, seal Sealer, id peer.ID, ns string, dist uint8) error {
	e, err := s.cache.LoadSurveyRequest(seal, id, ns, dist)
	if err != nil {
		return err
	}

	return s.Send(ctx, e, s.conn.LocalAddr())
}

func (s *Socket) SendResponse(seal Sealer, h Host, to net.Addr, ns string) error {
	e, err := s.cache.LoadResponse(seal, h, ns)
	if err != nil {
		return err
	}

	return s.conn.Send(ctxutil.C(s.done), e, to)
}

func (s *Socket) SendSurveyResponse(seal Sealer, h Host, ns string) error {
	e, err := s.cache.LoadResponse(seal, h, ns)
	if err != nil {
		return err
	}

	return s.Send(ctxutil.C(s.done), e, s.conn.LocalAddr())
}

func (s *Socket) Subscribe(ns string, limit int) (<-chan peer.AddrInfo, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		return closedChan, func() {}
	default:
	}

	ss, ok := s.subs[ns]
	if !ok {
		ss = make(subscriberSet)
		s.subs[ns] = ss
	}

	var (
		once sync.Once
		ch   = make(chan peer.AddrInfo, 16) // arbitrary buf size
	)

	cancel := func() {
		once.Do(func() {
			defer close(ch)

			s.mu.Lock()
			defer s.mu.Unlock()

			if ss.Remove(ch) {
				delete(s.subs, ns)
			}
		})
	}

	ss.Add(ch, limiter(limit, cancel))

	return ch, cancel
}

func (s *Socket) tickloop() {
	for t := range s.tick.C {
		s.mu.Lock()

		s.time = t
		for ns, deadline := range s.advt {
			if t.After(deadline) {
				delete(s.advt, ns)
			}
		}

		s.mu.Unlock()
	}
}

func (s *Socket) serve(h RequestHandler) {
	var (
		r    Record
		addr net.Addr
		err  error
	)

	for {
		if addr, err = s.conn.Scan(&r); err == nil {
			err = s.handle(h, r, addr)
		}

		if err != nil && !errors.Is(err, ErrIgnore) {
			select {
			case <-s.done:
				return
			default:
			}

			s.handleError(s, err)

			// socket closed?
			if ne, ok := err.(net.Error); ok && !ne.Timeout() {
				defer s.Close()
				return
			}
		}
	}
}

func (s *Socket) handle(h RequestHandler, r Record, addr net.Addr) error {
	ns, err := r.Namespace()
	if err != nil {
		return ProtocolError{
			Message: "failed to read namespace from record",
			Cause:   err,
		}
	}

	// Packet is already validated.  It's either a response, or some sort of request.
	switch r.Type() {
	case TypeResponse:
		return s.dispatch(Response{
			Record: r,
			NS:     ns,
			From:   addr,
		})

	case TypeRequest, TypeSurvey:
		if s.tracking(ns) {
			err = h(Request{
				Record: r,
				NS:     ns,
				From:   addr,
			})
		}

		return err

	default:
		return ProtocolError{
			Message: "invalid packet",
			Cause:   fmt.Errorf("unknown type: %s", r.Type()),
		}
	}
}

func (s *Socket) tracking(ns string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.advt[ns]
	return ok
}

func (s *Socket) dispatch(r Response) error {
	var info peer.AddrInfo
	if err := r.Bind(&info); err != nil {
		return err
	}

	// release any subscriptions that have reached their limit.
	var done []func()
	defer func() {
		for _, release := range done {
			release()
		}
	}()

	s.mu.RLock()
	defer s.mu.RUnlock()

	if ss, ok := s.subs[r.NS]; ok {
		for sub, lim := range ss {
			select {
			case sub.Out <- info:
				if lim.Decr() {
					done = append(done, lim.cancel)
				}

			default:
			}
		}
	}

	return nil
}

type subscriberSet map[subscriber]*resultLimiter

type resultLimiter struct {
	remaining int32
	cancel    func()
}

func limiter(limit int, cancel func()) (l *resultLimiter) {
	if limit > 0 {
		l = &resultLimiter{
			remaining: int32(limit),
			cancel:    cancel,
		}
	}

	return
}

func (l *resultLimiter) Decr() bool {
	return l != nil && atomic.AddInt32(&l.remaining, -1) == 0
}

type subscriber struct{ Out chan<- peer.AddrInfo }

func (ss subscriberSet) Add(s chan<- peer.AddrInfo, l *resultLimiter) {
	ss[subscriber{s}] = l
}

func (ss subscriberSet) Remove(s chan<- peer.AddrInfo) (empty bool) {
	delete(ss, subscriber{s})
	return len(ss) == 0
}

var closedChan = make(chan peer.AddrInfo)
