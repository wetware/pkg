package discover

import (
	"context"
	"net"
	"time"

	ww "github.com/lthibault/wetware/pkg"
	"github.com/pkg/errors"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/whyrusleeping/mdns"
)

const defaultTimeout = time.Second * 2

func init() {
	// logs produce false-positive errors.
	mdns.DisableLogging = true
}

// MDNS discovers bootstrap peers through multicast DNS (RFC 6762)
type MDNS struct {
	Namespace string
	Interface *net.Interface

	// Beacon stuff.  Will be uninitialized until a call to Start.
	server interface{ Shutdown() error }
}

// DiscoverPeers queries MDNS.
func (d MDNS) DiscoverPeers(ctx context.Context, opt ...Option) (<-chan peer.AddrInfo, error) {
	var p Param
	if err := p.Apply(opt); err != nil {
		return nil, err
	}

	out := make(chan peer.AddrInfo, 1)
	entries := make(chan *mdns.ServiceEntry, 8)

	go func() {
		if err := mdns.Query(&mdns.QueryParam{
			Timeout:             getTimeout(ctx),
			Service:             d.namespace(),
			Entries:             entries,
			Interface:           d.Interface,
			WantUnicastResponse: true,
		}); err != nil {
			p.Log().WithError(err).Debug("mdns query failed")
			// return nil, errors.Wrap(err, "mdns query")
		}
	}()

	go func() {
		defer close(out)

		remaining := p.Limit

		for {
			select {
			case entry := <-entries:
				info, err := d.handleEntry(entry)
				if err != nil {
					p.Log().WithError(err).Debug("failed to handle MDNS entry")
					continue
				}

				select {
				case out <- info:
					if p.isLimited() {
						if remaining--; remaining == 0 {
							return
						}
					}
				case <-ctx.Done():
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, ctx.Err()
}

// Start an MDNS server that responds to queries in the background.
func (d *MDNS) Start(s Service) error {
	p, err := getDialableListenAddrs(s)
	if err != nil {
		return err
	}

	zone, err := mdns.NewMDNSService(s.ID().Pretty(),
		d.namespace(),
		"", "",
		p.Port(), p.IPs(), // these fields are required by MDNS but ignored by ww
		marshalTxtRecord(s)) // peer.ID and multiaddrs are stored here
	if err != nil {
		return err
	}

	d.server, err = mdns.NewServer(&mdns.Config{
		Zone:  zone,
		Iface: d.Interface,
	})

	return err
}

// Close the server.  Panics if ListenAndServe was not previously called.
func (d MDNS) Close() error {
	return d.server.Shutdown()
}

func (d MDNS) handleEntry(e *mdns.ServiceEntry) (info peer.AddrInfo, err error) {
	if info.ID, err = peer.IDB58Decode(e.InfoFields[0]); err != nil {
		return
	}

	info.Addrs = make([]multiaddr.Multiaddr, len(e.InfoFields)-1) // 0th item is peer.ID
	for i, s := range e.InfoFields[1:] {
		if info.Addrs[i], err = multiaddr.NewMultiaddr(s); err != nil {
			break
		}
	}

	return
}

func getDialableListenAddrs(s Service) (p payload, err error) {
	var as []multiaddr.Multiaddr
	if as, err = s.Network().InterfaceListenAddresses(); err != nil {
		return nil, err
	}

	for _, addr := range as {
		na, err := manet.ToNetAddr(addr)
		if err != nil {
			continue
		}

		switch a := na.(type) {
		case *net.TCPAddr:
			p = append(p, address{IP: a.IP, Port: a.Port})
		case *net.UDPAddr:
			p = append(p, address{IP: a.IP, Port: a.Port})
		}
	}

	if len(p) == 0 {
		return nil, errors.New("failed to resolve external addr from service")
	}

	return p, nil
}

func (d MDNS) namespace() string {
	if d.Namespace != "" {
		return d.Namespace
	}

	return ww.DefaultNamespace
}

type address struct {
	IP   net.IP
	Port int
}

type payload []address

func (p payload) Port() int {
	return p[0].Port
}

func (p payload) IPs() []net.IP {
	return []net.IP{p[0].IP}
}

func marshalTxtRecord(s Service) []string {
	out := []string{s.ID().String()}

	for _, addr := range s.Network().ListenAddresses() {
		out = append(out, addr.String())
	}

	return out
}

func getTimeout(ctx context.Context) time.Duration {
	if t, ok := ctx.Deadline(); ok {
		return t.Sub(time.Now())
	}

	return defaultTimeout
}
