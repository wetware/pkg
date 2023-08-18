package survey

import (
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/record"

	api "github.com/wetware/pkg/api/boot"
	"github.com/wetware/pkg/boot/socket"
)

// Surveyor discovers peers through a surveyor/respondent multicast
// protocol.
type Surveyor struct {
	host host.Host
	sock *socket.Socket
}

// New surveyor.  The supplied PacketConn SHOULD be bound to a multicast
// group.  Use of JoinMulticastGroup to construct conn is RECOMMENDED.
func New(h host.Host, sock *socket.Socket) *Surveyor {
	s := &Surveyor{
		host: h,
		sock: sock,
	}

	go s.sock.Bind(s.handler())

	return s
}

func (s *Surveyor) Close() error {
	return s.sock.Close()
}

func (s *Surveyor) handler() socket.RequestHandler {
	return func(r socket.Request) error {
		id, err := r.Peer()
		if err != nil {
			return socket.ProtocolError{
				Message: "invalid ID in request",
				Cause:   err,
				Packet:  api.Packet(r.Record),
			}
		}

		// distance too large?
		if ignore(s.host.ID(), id, r.Distance()) {
			return socket.ErrIgnore
		}

		return s.sock.SendSurveyResponse(s.sealer(), s.host, r.NS)
	}
}

func (s *Surveyor) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (ttl time.Duration, err error) {
	if len(s.host.Addrs()) == 0 {
		return 0, errors.New("no listen addrs")
	}

	var opts = discovery.Options{Ttl: peerstore.TempAddrTTL}
	if err = opts.Apply(opt...); err != nil {
		return
	}

	if err = s.sock.Track(ns, opts.Ttl); err == nil {
		ttl = opts.Ttl
	}

	return
}

func (s *Surveyor) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	var opts discovery.Options
	if err := opts.Apply(opt...); err != nil {
		return nil, err
	}

	out, cancel := s.sock.Subscribe(ns, opts.Limit)
	go func() {
		defer cancel()

		select {
		case <-ctx.Done():
		case <-s.sock.Done():
		}
	}()

	// Send multicast request.
	err := s.sock.SendSurveyRequest(ctx, s.sealer(), s.host.ID(), ns, distance(opts))
	if err != nil {
		cancel()
		return nil, err
	}

	return out, nil
}

func (s *Surveyor) sealer() socket.Sealer {
	return func(r record.Record) (*record.Envelope, error) {
		return record.Seal(r, privkey(s.host))
	}
}

func privkey(h host.Host) crypto.PrivKey {
	return h.Peerstore().PrivKey(h.ID())
}

func ignore(local, remote peer.ID, d uint8) bool {
	return xor(local, remote)>>uint32(d) != 0
}

func xor(id1, id2 peer.ID) uint32 {
	xored := make([]byte, 4)
	for i := 0; i < 4; i++ {
		xored[i] = id1[len(id1)-i-1] ^ id2[len(id2)-i-1]
	}

	return binary.BigEndian.Uint32(xored)
}
