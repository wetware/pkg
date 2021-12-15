package start

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/boot"
	"go.uber.org/fx"
)

type bootContext struct {
	fx.Out

	Advertiser discovery.Advertiser
	Discoverer discovery.Discoverer
	Service    boot.Service
}

func bindBootContext(c *cli.Context, handler boot.Handler) (b bootContext, err error) {
	defer func() {
		if exc, ok := recover().(error); ok {
			err = exc // abruptly return the paniced error
		}
	}()

	b = bootContext{
		Advertiser: advertiser(c),
		Discoverer: &boot.Context{
			Strategy: &boot.ScanSubnet{
				Port:    port(c), // may panic
				CIDR:    cidr(c), // may panic
				Handler: handler,
			},
		},
		Service: boot.Service{
			Advertiser: b.Advertiser,
			Discoverer: b.Discoverer,
		},
	}

	return
}

func port(c *cli.Context) (port int) {
	selector := scanAddr(c.String("discovery")).String() // cidr:port

	// get the last segment of a ':'-separated path, defaulting to "".
	var portNum string
	for _, portNum = range strings.SplitN(selector, ":", 2) {
		port, _ = strconv.Atoi(portNum)
	}

	return
}

func cidr(c *cli.Context) boot.CIDR {
	selector := scanAddr(c.String("discovery")).String() // cidr:port
	subnet := strings.TrimSuffix(selector, fmt.Sprintf(":%d", port(c)))

	return boot.CIDR{
		Subnet: subnet,
	}
}

// func bind(c *cli.Context, a func() discovery.Advertiser, s boot.ScanStrategy) *cli.Context {
// 	c = withAdvertiser(c, a)
// 	c = withDiscoverer(c, newScanner(func() *boot.Context {
// 		return &boot.Context{
// 			Net:      new(net.Dialer),
// 			Strategy: s,
// 		}
// 	}))
// 	return c

// }

func advertiser(c *cli.Context) discovery.Advertiser {
	return c.Context.Value(newAdvertiser(nil)).(discovery.Advertiser)
}

// func strategy(c *cli.Context) boot.ScanStrategy {
// 	return c.Context.Value(newScanner(nil)).(boot.ScanStrategy)
// }

// func withAdvertiser(c *cli.Context, a func() discovery.Advertiser) *cli.Context {
// 	c.Context = context.WithValue(c.Context, newAdvertiser(nil), a)
// 	return c
// }

// func withDiscoverer(c *cli.Context, s newScanner) *cli.Context {
// 	c.Context = context.WithValue(c.Context, newScanner(nil), s)
// 	return c
// }

type newAdvertiser func() discovery.Advertiser

func (advertiser newAdvertiser) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	return advertiser().Advertise(ctx, ns, opt...)
}

// type newScanner func() *boot.Context

// func bindScanOptions(opts *discovery.Options, opt []discovery.Option) error {
// 	return opts.Apply(opt...)
// }

// func (scanner newScanner) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
// 	var opts = discovery.Options{
// 		Limit: 1,
// 	}

// 	if err := bindScanOptions(&opts, opt); err != nil {
// 		return nil, nil
// 	}

// 	agent := make(chan peer.AddrInfo, 1)
// 	return agent, scanner.bind(ctx, agent, scanAddr(ns), &opts)
// }

// func (scanner newScanner) bind(ctx context.Context, out chan<- peer.AddrInfo, addr net.Addr, opts *discovery.Options) error {
// 	var rec peer.PeerRecord

// 	scan := scanner()
// 	if _, err := scan.Strategy.Scan(ctx, scan.Net, &rec); err != nil {
// 		return err
// 	}

// 	select {
// 	case out <- peer.AddrInfo{ID: rec.PeerID, Addrs: rec.Addrs}:
// 		return nil

// 	case <-ctx.Done():
// 		return ctx.Err()
// 	}
// }

type scanAddr string

func (addr scanAddr) Network() string {
	u, err := url.Parse(string(addr))
	if err != nil {
		panic(fmt.Errorf("invalid namespace: %w", err))
	}

	return u.Scheme
}

func (addr scanAddr) String() string {
	u, err := url.Parse(string(addr))
	if err != nil {
		panic(fmt.Errorf("invalid namespace: %w", err))
	}

	return u.Host
}
