package discover

import (
	"context"
	"net"

	ww "github.com/lthibault/wetware/pkg"
	"github.com/pkg/errors"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/whyrusleeping/mdns"
)

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
func (d MDNS) DiscoverPeers(ctx context.Context) ([]peer.AddrInfo, error) {
	// TODO:  implement discover.DiscoverOpt to set the desired number of boot peers.
	//		  In many cases, it's helpful to get n == ww.LowWater bootstrap peers.

	entries := make(chan *mdns.ServiceEntry, 1)

	if err := mdns.Query(&mdns.QueryParam{
		Service:             d.namespace(),
		Entries:             entries,
		Interface:           d.Interface,
		WantUnicastResponse: true,
	}); err != nil {
		return nil, errors.Wrap(err, "mdns query")
	}

	select {
	case entry := <-entries:
		return d.handleEntry(entry)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
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

func (d MDNS) handleEntry(e *mdns.ServiceEntry) ([]peer.AddrInfo, error) {
	id, err := peer.IDB58Decode(e.InfoFields[0])
	if err != nil {
		return nil, err
	}

	as := make([]multiaddr.Multiaddr, len(e.InfoFields)-1)
	for i, a := range e.InfoFields[1:] {
		if as[i], err = multiaddr.NewMultiaddr(a); err != nil {
			return nil, err
		}
	}

	return []peer.AddrInfo{
		{ID: id, Addrs: as},
	}, nil
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
