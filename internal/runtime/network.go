package runtime

import (
	"context"
	"net"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	disc "github.com/libp2p/go-libp2p-discovery"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/pex"
)

func bindNetwork() fx.Option {
	return fx.Provide(
		bindCluster,
		bindPubSub,
		bindDiscovery,
		bindCrawler)
}

type clusterConfig struct {
	fx.In

	Logger log.Logger
	PubSub *pubsub.PubSub

	Lifecycle fx.Lifecycle
}

func bindCluster(c *cli.Context, config clusterConfig) (*cluster.Node, error) {
	node, err := cluster.New(c.Context, config.PubSub,
		cluster.WithLogger(config.Logger),
		cluster.WithNamespace(c.String("ns")))

	if err == nil {
		config.Lifecycle.Append(closer(node))
	}

	return node, err
}

func bindPubSub(c *cli.Context, h host.Host, d discovery.Discovery) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(c.Context, h, pubsub.WithDiscovery(d))
}

type discoveryConfig struct {
	fx.In

	Logger    log.Logger
	Host      host.Host
	Datastore ds.Batching
	DHT       *dual.DHT

	Crawler    boot.Crawler
	Beacon     boot.Beacon
	Supervisor *suture.Supervisor

	Lifecycle fx.Lifecycle
}

func bindDiscovery(c *cli.Context, config discoveryConfig) (discovery.Discovery, error) {
	var token suture.ServiceToken
	config.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			token = config.Supervisor.Add(config.Beacon)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return config.Supervisor.RemoveAndWait(token, timeout(ctx))
		},
	})

	// Wrap the bootstrap discovery service in a peer sampling service.
	px, err := pex.New(c.Context, config.Host,
		pex.WithLogger(config.Logger),
		pex.WithDatastore(config.Datastore),
		pex.WithDiscovery(struct {
			discovery.Discoverer
			discovery.Advertiser
		}{
			Discoverer: config.Crawler,
			Advertiser: config.Beacon,
		}))

	// If the namespace matches the cluster pubsub topic,
	// fetch peers from PeX, which itself will fall back
	// on the bootstrap service 'p'.
	return boot.Cache{
		Match: exactly(c.String("ns")),
		Cache: px,
		Else:  disc.NewRoutingDiscovery(config.DHT),
	}, err
}

func exactly(match string) func(string) bool {
	return func(s string) bool {
		return match == s
	}
}

func bindCrawler(c *cli.Context) boot.Crawler {
	return boot.Crawler{
		Net: new(net.Dialer),
		Strategy: &boot.ScanSubnet{
			Net:    "tcp",
			Port:   8822,
			Subnet: boot.Subnet{CIDR: "127.0.0.1/24"}, // XXX
		},
	}
}

func timeout(ctx context.Context) time.Duration {
	if t, ok := ctx.Deadline(); ok {
		return time.Until(t)
	}

	return time.Second * 5
}

// // ...

// type bootContext struct {
// 	fx.Out

// 	Advertiser discovery.Advertiser
// 	Discoverer discovery.Discoverer
// 	Service    discovery.Discovery
// }

// func bindNetwork(c *cli.Context, scanner boot.Scanner) (b bootContext, err error) {
// 	defer func() {
// 		if exc, ok := recover().(error); ok {
// 			err = exc // abruptly return the panicked error
// 		}
// 	}()

// 	b = bootContext{
// 		Advertiser: advertiser(c),
// 		Discoverer: netscan(c, scanner),
// 		Service:    service(&b),
// 	}

// 	return
// }

// func netscan(c *cli.Context, scanner boot.Scanner) discovery.Discoverer {
// 	return &boot.DiscoveryService{
// 		Strategy: &boot.ScanSubnet{
// 			Net:     "tcp",
// 			Port:    port(c), // may panic
// 			Subnet:  cidr(c), // may panic
// 			Scanner: scanner,
// 		},
// 	}
// }

// func service(bcx *bootContext) discovery.Discovery {
// 	return struct {
// 		discovery.Advertiser
// 		discovery.Discoverer
// 	}{
// 		Advertiser: bcx.Advertiser,
// 		Discoverer: bcx.Discoverer,
// 	}
// }

// func port(c *cli.Context) (port int) {
// 	selector := scanAddr(c.String("discovery")).String() // cidr:port

// 	// get the last segment of a ':'-separated path, defaulting to "".
// 	var portNum string
// 	for _, portNum = range strings.SplitN(selector, ":", 2) {
// 		port, _ = strconv.Atoi(portNum)
// 	}

// 	return
// }

// func cidr(c *cli.Context) boot.Subnet {
// 	return boot.Subnet{
// 		CIDR: c.String("subnet"),
// 	}
// }

// // func bind(c *cli.Context, a func() discovery.Advertiser, s boot.ScanStrategy) *cli.Context {
// // 	c = withAdvertiser(c, a)
// // 	c = withDiscoverer(c, newScanner(func() *boot.Context {
// // 		return &boot.Context{
// // 			Net:      new(net.Dialer),
// // 			Strategy: s,
// // 		}
// // 	}))
// // 	return c

// // }

// func advertiser(c *cli.Context) discovery.Advertiser {
// 	return c.Context.Value(newAdvertiser(nil)).(discovery.Advertiser)
// }

// // func strategy(c *cli.Context) boot.ScanStrategy {
// // 	return c.Context.Value(newScanner(nil)).(boot.ScanStrategy)
// // }

// // func withAdvertiser(c *cli.Context, a func() discovery.Advertiser) *cli.Context {
// // 	c.Context = context.WithValue(c.Context, newAdvertiser(nil), a)
// // 	return c
// // }

// // func withDiscoverer(c *cli.Context, s newScanner) *cli.Context {
// // 	c.Context = context.WithValue(c.Context, newScanner(nil), s)
// // 	return c
// // }

// type newAdvertiser func() discovery.Advertiser

// func (advertiser newAdvertiser) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
// 	return advertiser().Advertise(ctx, ns, opt...)
// }

// // type newScanner func() *boot.Context

// // func bindScanOptions(opts *discovery.Options, opt []discovery.Option) error {
// // 	return opts.Apply(opt...)
// // }

// // func (scanner newScanner) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
// // 	var opts = discovery.Options{
// // 		Limit: 1,
// // 	}

// // 	if err := bindScanOptions(&opts, opt); err != nil {
// // 		return nil, nil
// // 	}

// // 	agent := make(chan peer.AddrInfo, 1)
// // 	return agent, scanner.bind(ctx, agent, scanAddr(ns), &opts)
// // }

// // func (scanner newScanner) bind(ctx context.Context, out chan<- peer.AddrInfo, addr net.Addr, opts *discovery.Options) error {
// // 	var rec peer.PeerRecord

// // 	scan := scanner()
// // 	if _, err := scan.Strategy.Scan(ctx, scan.Net, &rec); err != nil {
// // 		return err
// // 	}

// // 	select {
// // 	case out <- peer.AddrInfo{ID: rec.PeerID, Addrs: rec.Addrs}:
// // 		return nil

// // 	case <-ctx.Done():
// // 		return ctx.Err()
// // 	}
// // }

// type scanAddr string

// func (addr scanAddr) Network() string {
// 	u, err := url.Parse(string(addr))
// 	if err != nil {
// 		panic(fmt.Errorf("invalid namespace: %w", err))
// 	}

// 	return u.Scheme
// }

// func (addr scanAddr) String() string {
// 	u, err := url.Parse(string(addr))
// 	if err != nil {
// 		panic(fmt.Errorf("invalid namespace: %w", err))
// 	}

// 	return u.Host
// }
