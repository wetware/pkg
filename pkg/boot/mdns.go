package boot

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
	// TODO:  implement boot.DiscoverOpt to set the desired number of boot peers.
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
	as, err := getDialableListenAddrs(s)
	if err != nil {
		return err
	}

	port := as[0].Port
	ips := make([]net.IP, len(as))
	for i, a := range as {
		ips[i] = a.IP
	}

	zone, err := mdns.NewMDNSService(s.ID().Pretty(),
		d.namespace(),
		"", "",
		port, ips,
		[]string{s.ID().Pretty()})
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
	mpeer, err := peer.IDB58Decode(e.Info)
	if err != nil {
		return nil, errors.Wrap(err, "decode b58")
	}

	maddr, err := manet.FromNetAddr(&net.TCPAddr{IP: e.AddrV4, Port: e.Port})
	if err != nil {
		return nil, errors.Wrap(err, "parse multiaddr")
	}

	return []peer.AddrInfo{
		{ID: mpeer, Addrs: []multiaddr.Multiaddr{maddr}},
	}, nil
}

func getDialableListenAddrs(s Service) (out []address, err error) {
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
			out = append(out, address{IP: a.IP, Port: a.Port})
		case *net.UDPAddr:
			out = append(out, address{IP: a.IP, Port: a.Port})
		}
	}

	if len(out) == 0 {
		return nil, errors.New("failed to resolve external addr from service")
	}

	return out, nil
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
