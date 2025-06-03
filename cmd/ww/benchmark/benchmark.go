package benchmark

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"runtime"

	//"runtime/pprof"
	"strconv"
	"sync"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	disc_util "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/urfave/cli/v2"

	core_api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/csp"
	csp_server "github.com/wetware/pkg/cap/csp/server"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/vat"
)

var meta tags

var flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "listen",
		Aliases: []string{"l"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip6/::0/udp/0/quic-v1"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringSliceFlag{
		Name:    "meta",
		Usage:   "metadata fields in key=value format",
		EnvVars: []string{"WW_META"},
	},
	&cli.Int64Flag{
		Name: "procs",
	},
	&cli.Int64Flag{
		Name: "total",
	},
	&cli.Int64Flag{
		Name: "yield",
	},
	&cli.Int64Flag{
		Name: "iters",
	},
}

//go:embed wasm/busy/busy.wasm
var busy []byte

type execArgs struct {
	args []string
	sess core_api.Session
}

func (a execArgs) Args() (capnp.TextList, error) {
	return csp.EncodeTextList(a.args)
}

func (a execArgs) Ppid() uint32 {
	return 1
}
func (a execArgs) Session() (core_api.Session, error) {
	return a.sess, nil
}

func Command() *cli.Command {
	return &cli.Command{
		Name:   "benchmark",
		Usage:  "benchmark the executor",
		Flags:  flags,
		Before: setup,
		Action: benchmark,
	}
}

func setup(c *cli.Context) error {
	deduped := make(map[routing.MetaField]struct{})
	for _, tag := range c.StringSlice("meta") {
		field, err := routing.ParseField(tag)
		if err != nil {
			return err
		}

		deduped[field] = struct{}{}
	}

	for tag := range deduped {
		meta = append(meta, tag)
	}

	return nil
}

func benchmark(c *cli.Context) error {
	procs := c.Int64("procs")
	total := c.Int64("total")
	yield := c.Int64("yield")
	iters := c.Int64("iters")
	if procs <= 0 || total <= 0 || yield < 0 || iters <= 0 {
		return errors.New("empty or invalid procs, total, yield or iters")
	}

	fmt.Printf(`{"procs": %d, "iters": %d, "total": %d, "yield": %d, "cores": %d}%s`,
		procs, iters, total, yield, runtime.GOMAXPROCS(0), "\n")

	ec := make(chan csp_server.Runtime, 1)
	sc := make(chan core_api.Session, 1)
	go serve(c, ec, sc)
	session := <-sc
	executor := <-ec
	args := execArgs{
		args: []string{
			strconv.FormatInt(total, 10),
			strconv.FormatInt(yield, 10),
		},
		sess: session,
	}
	cid := executor.Cache.ExposedPut(busy)
	// Cache the WASM compilation
	p1, err := executor.ExposedExec(c.Context, cid, busy, args)
	if err != nil {
		panic(err)
	}
	csp.Proc(p1).Wait(c.Context)

	ms_per_proc := make([]int64, iters*procs)
	ms_per_iter := make([]int64, iters)

	startTotal := time.Now()
	for i := int64(0); i < iters; i++ {
		startIter := time.Now()
		fmt.Printf("run iteration %d\n", i)
		var wg sync.WaitGroup
		for j := int64(0); j < procs; j++ {
			wg.Add(1)
			go func(k int64) {
				defer wg.Done()
				startProc := time.Now()
				p, err := executor.ExposedExec(c.Context, cid, busy, args)
				if err != nil {
					panic(err)
				}
				csp.Proc(p).Wait(c.Context)
				endProc := time.Now()
				ms_per_proc[i*procs+k] = endProc.Sub(startProc).Milliseconds()
			}(j)
		}
		wg.Wait()

		endIter := time.Now()
		ms_per_iter[i] = endIter.Sub(startIter).Milliseconds()
	}
	endTotal := time.Now()
	//pprof.StopCPUProfile()
	ms_total := endTotal.Sub(startTotal).Milliseconds()

	avg_per_proc := int64(0)
	avg_per_iter := int64(0)
	for i := int64(0); i < iters; i++ {
		avg_per_iter += ms_per_iter[i]
		for j := int64(0); j < procs; j++ {
			avg_per_proc += ms_per_proc[i*procs+j]
		}
	}
	avg_per_proc /= iters * procs
	avg_per_iter /= iters
	fmt.Printf(`{
		"procs": %d,
		"iters": %d,
		"total": %d,
		"yield": %d,
		"cores": %d,
		"avg_ms_per_proc": %d,
		"avg_ms_per_iter": %d,
		"total_ms": %d,
	%s}%s`,
		procs, iters, total, yield, runtime.GOMAXPROCS(0),
		avg_per_proc, avg_per_iter, ms_total, "\r", "\n")

	return nil
}

func serve(c *cli.Context, ec chan csp_server.Runtime, sc chan core_api.Session) error {
	h, err := vat.ListenP2P(c.StringSlice("listen")...)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer h.Close()

	dht, err := vat.NewDHT(c.Context, h, c.String("ns"))
	if err != nil {
		return fmt.Errorf("dht: %w", err)
	}
	defer dht.Close()

	bootstrap, err := newBootstrap(c, h)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer bootstrap.Close()

	return vat.Config{
		NS:        c.String("ns"),
		Host:      routedhost.Wrap(h, dht),
		Bootstrap: bootstrap,
		Ambient:   ambient(dht),
		Meta:      meta,
		Auth:      auth.AllowAll,
	}.Serve(c.Context, ec, sc, h)
}

func newBootstrap(c *cli.Context, h local.Host) (_ boot.Service, err error) {
	// use discovery service?
	if len(c.StringSlice("peer")) == 0 {
		serviceAddr := c.String("discover")
		return boot.ListenString(h, serviceAddr)
	}

	// fast path; direct dial a peer
	maddrs := make([]ma.Multiaddr, len(c.StringSlice("peer")))
	for i, s := range c.StringSlice("peer") {
		if maddrs[i], err = ma.NewMultiaddr(s); err != nil {
			return
		}
	}

	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	return boot.StaticAddrs(infos), err
}

func ambient(dht *dual.DHT) discovery.Discovery {
	return disc_util.NewRoutingDiscovery(dht)
}

type tags []routing.MetaField

func (tags tags) Prepare(h pulse.Heartbeat) error {
	if err := h.SetMeta(tags); err != nil {
		return err
	}

	// hostname may change over time
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	return h.SetHost(host)
}
