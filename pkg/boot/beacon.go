package boot

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jpillora/backoff"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/record"
	"github.com/lthibault/log"
	"golang.org/x/crypto/sha3"
)

type Beacon struct {
	once   sync.Once
	Logger log.Logger

	Envelope *record.Envelope
	Addr     string

	// atomicBeaconState
	//  : cq        chan struct{}
	//  | advertise chan<- *discovery.Options
	//  ;
	state atomicBeaconState
}

func (b *Beacon) String() string { return "casm.boot.beacon" }

func (b *Beacon) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"addr": b.Addr,
	}
}

func (b *Beacon) Serve(ctx context.Context) error {
	b.once.Do(func() {
		if b.Logger == nil {
			b.Logger = log.New(log.WithLevel(log.FatalLevel))
		}
	})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		namespaces = make(map[string]time.Time)
		advertise  = make(chan *discovery.Options)
		knock      = make(chan knockRequest)
		cherr      = make(chan error, 1)
	)
	b.state.Set(ctx, advertise)
	defer b.state.Reset()

	b.Logger.Debug("beacon started")
	defer close(knock)
	defer close(advertise)

	conn, err := b.listen(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		var (
			k   knockRequest
			buf [bufsize]byte
		)

		for {
			n, addr, err := conn.ReadFromUDP(buf[:])
			if err != nil {
				return
			}

			b.Logger.WithField("size", n).Trace("got message from: %s", addr)

			err = k.Knock.UnmarshalBinary(buf[:n])
			if err != nil {
				b.Logger.WithError(err).
					WithField("from", addr.String()).
					Debug("error reading payload")
				continue
			}

			select {
			case knock <- knockRequest{Knock: k.Knock, Dialback: addr}:
			case <-ctx.Done():
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			for ns, dl := range namespaces {
				if dl.After(t) {
					continue
				}

				delete(namespaces, ns)
			}

		case o := <-advertise:
			namespaces[namespace(o)] = time.Now().Add(o.Ttl)
			signal(o)

		case k := <-knock:
			for ns := range namespaces {
				if !k.Matches(ns) {
					continue
				}

				b.Logger.With(k).WithField("ns", ns).Trace("matched")

				if err := b.reply(ctx, conn, k.Dialback); err != nil {
					return err
				}
			}

		case err := <-cherr:
			return err

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Advertise the specified namespace with a default TTL of 24h.
func (b *Beacon) Advertise(ctx context.Context, ns string, opts ...discovery.Option) (time.Duration, error) {
	// This MUST be loaded exactly once per call,
	// else a race condition can occur between calls.
	state, err := b.state.Load(ctx)
	if err != nil {
		return 0, err
	}

	o := &discovery.Options{
		Ttl: peerstore.PermanentAddrTTL,
		Other: map[interface{}]interface{}{
			keyNS{}:     ns,
			keySignal{}: make(chan struct{}),
		},
	}

	if err := o.Apply(opts...); err != nil {
		return 0, err
	}

	select {
	case state.advertise <- o:
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-state.cq:
		return 0, fmt.Errorf("closing")
	}

	select {
	case <-wait(o):
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-state.cq:
		return 0, fmt.Errorf("closing")
	}

	return o.Ttl, nil
}

func (b *Beacon) reply(ctx context.Context, conn *net.UDPConn, addr net.Addr) error {
	bs, err := b.Envelope.Marshal()
	if err != nil {
		return err
	}

	dl, _ := ctx.Deadline()
	if err = conn.SetWriteDeadline(dl); err != nil {
		return err
	}

	_, err = conn.WriteTo(bs, addr)
	return err
}

func (b *Beacon) listen(ctx context.Context) (conn *net.UDPConn, err error) {
	var addr *net.UDPAddr
	addr, err = net.ResolveUDPAddr("udp4", b.Addr)
	if err != nil {
		return
	}

	conn, err = net.ListenUDP("udp4", addr)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	for _, set := range []func() error{
		func() error { return conn.SetReadBuffer(bufsize) },
		func() error { return conn.SetWriteBuffer(bufsize) },
	} {
		if err = set(); err != nil {
			break
		}
	}

	return
}

type Knock struct {
	Nonce [8]byte  // MUST originate from cryptographically secure PRNG
	Hash  [64]byte // SHA-3
}

func NewKnock(ns string) (k Knock, err error) {
	if _, err = rand.Read(k.Nonce[:]); err == nil {
		h := sha3.New512()
		io.Copy(h, bytes.NewReader(k.Nonce[:]))
		io.Copy(h, strings.NewReader(ns))
		copy(k.Hash[:], h.Sum(nil))
	}

	return
}

func (k Knock) Bytes() []byte {
	return append(k.Nonce[:], k.Hash[:]...)
}

func (k *Knock) UnmarshalBinary(b []byte) error {
	if len(b) != len(k.Hash)+len(k.Nonce) {
		return fmt.Errorf("len(b) != 72")
	}

	copy(k.Nonce[:], b)
	copy(k.Hash[:], b[8:])
	return nil
}

func (k Knock) Matches(ns string) bool {
	h := sha3.New512()
	io.Copy(h, bytes.NewReader(k.Nonce[:]))
	io.Copy(h, strings.NewReader(ns))
	return bytes.Equal(k.Hash[:], h.Sum(nil))
}

type keyNS struct{}

func namespace(o *discovery.Options) string {
	return o.Other[keyNS{}].(string)
}

type keySignal struct{}

func signal(o *discovery.Options) {
	close(o.Other[keySignal{}].(chan struct{}))
}

func wait(o *discovery.Options) <-chan struct{} {
	return o.Other[keySignal{}].(chan struct{})
}

type atomicBeaconState atomic.Value

func (a *atomicBeaconState) Load(ctx context.Context) (beaconState, error) {
	var b = backoff.Backoff{
		Factor: 2,
		Min:    time.Millisecond,
		Max:    time.Millisecond * 512,
	}

	for {
		state, ok := (*atomic.Value)(a).Load().(beaconState)
		if ok || state.cq != nil {
			select {
			case <-state.cq: // restarting
			default:
				return state, nil
			}
		}

		select {
		case <-time.After(b.Duration()):
		case <-ctx.Done():
			return beaconState{}, ctx.Err()
		}
	}
}

func (a *atomicBeaconState) Set(ctx context.Context, advertise chan<- *discovery.Options) {
	(*atomic.Value)(a).Store(beaconState{
		cq:        ctx.Done(),
		advertise: advertise,
	})
}

func (a *atomicBeaconState) Reset() { (*atomic.Value)(a).Store(beaconState{}) }

type beaconState struct {
	cq        <-chan struct{}
	advertise chan<- *discovery.Options
}
