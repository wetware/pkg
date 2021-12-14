package boot

import (
	"context"
	"encoding/binary"
	insecure "math/rand"
	"net"
	"time"

	"github.com/jbenet/goprocess"
	goprocessctx "github.com/jbenet/goprocess/context"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	"github.com/lthibault/log"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/packet"
)

var defaultListenConfig net.ListenConfig

type PacketListener interface {
	ListenPacket(ctx context.Context, network, addr string) (net.PacketConn, error)
}

type PortListener struct {
	Logger log.Logger

	packet.Endpoint
	Network PacketListener
}

func (pl PortListener) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// We are effectively "serving" the endpoint over the network.  Listen on
	// the address it provides.
	conn, err := packetNet{pl.Network}.ListenPacket(ctx,
		pl.Endpoint.Addr.Network(),
		pl.Endpoint.Addr.String())
	if err != nil {
		return err
	}
	defer conn.Close()

	// Get network/addr from the active conn.  This will give us the resolved
	// endpoint address.
	log := pl.Logger.
		WithField("net", conn.LocalAddr().Network()).
		WithField("conn", conn.LocalAddr().String())

	// Answer all packets until 'conn' is closed.
	cherr := make(chan error, 1)
	go func() {
		for {
			if err = boot.Answer(ctx, pl.Endpoint, conn); err != nil {
				cherr <- err
				return
			}

			log.Trace("handled packet")
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-cherr:
		return err
	}
}

func (pl PortListener) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	var o = discovery.Options{Ttl: time.Hour}
	err := o.Apply(opt...)
	return o.Ttl, err
}

type PortKnocker struct {
	Logger      log.Logger
	Network     PacketListener
	Port        int
	Subnet      *net.IPNet
	RequestBody *record.Envelope
}

func (pk PortKnocker) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := packetNet{pk.Network}.ListenPacket(ctx, "udp4", ":0") // all interfaces, arbitrary port
	if err != nil {
		return nil, err
	}

	out := make(chan peer.AddrInfo, 8)
	goprocessctx.WithContext(ctx).Go(pk.knock(conn, out)).SetTeardown(func() error {
		defer cancel()

		return conn.Close()
	})

	return out, nil
}

func (pk PortKnocker) knock(conn net.PacketConn, out chan<- peer.AddrInfo) func(goprocess.Process) {
	return func(proc goprocess.Process) {
		defer close(out)

		var (
			// Convert IPNet struct mask and address to uint32.
			// Network is BigEndian.
			mask  = binary.BigEndian.Uint32(pk.Subnet.Mask)
			begin = binary.BigEndian.Uint32(pk.Subnet.IP)
			end   = (begin & mask) | (mask ^ 0xffffffff) // final address

			// Each IP will be masked with the nonce before knocking.
			// This effectively randomizes the search.
			randmask = insecure.Uint32() & (mask ^ 0xffffffff)
		)

		var (
			ctx  = goprocessctx.OnClosingContext(proc)
			req  = casm.New(pk.RequestBody)
			addr = net.UDPAddr{
				Port: pk.Port,
				IP:   make(net.IP, 4),
			}
		)

		// loop through CIDR as unt32
		for i := begin; i <= end; i++ {
			// Populate the current IP address.
			binary.BigEndian.PutUint32(addr.IP, i^randmask)

			// Skip X.X.X.0 and X.X.X.255
			if i^randmask == begin || i^randmask == end {
				continue
			}

			info, err := boot.Knock(ctx, packet.Endpoint{
				Register: req,
				Addr:     &addr,
			}, conn)

			if err != nil {
				pk.Logger.WithError(err).Debugf("knock failed for %s", addr.String())
				continue
			}

			select {
			case out <- info:
			case <-ctx.Done():
				return
			}
		}
	}
}

type packetNet struct{ PacketListener }

func (n packetNet) ListenPacket(ctx context.Context, network, addr string) (net.PacketConn, error) {
	if n.PacketListener == nil {
		n.PacketListener = &defaultListenConfig
	}

	return n.PacketListener.ListenPacket(ctx, network, addr)
}
