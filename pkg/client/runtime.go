package client

import (
	"context"
	"time"

	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/pnet"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/config"

	ctxutil "github.com/lthibault/wetware/internal/util/ctx"
	hostutil "github.com/lthibault/wetware/internal/util/host"
	discover "github.com/lthibault/wetware/pkg/discover"
	"github.com/lthibault/wetware/pkg/internal/p2p"
	"github.com/lthibault/wetware/pkg/internal/runtime"
)

// Config contains user-supplied parameters used by Dial.
type Config struct {
	log log.Logger
	ns  string
	psk pnet.PSK
	ds  datastore.Batching

	d          discover.Strategy
	queryLimit int
}

func (cfg Config) assemble(ctx context.Context, c *Client) {
	c.app = fx.New(
		fx.NopLogger,
		fx.Populate(c),
		fx.Provide(
			cfg.options,
			p2p.New,
			newClient,
		),
		runtime.ClientEnv(),
	)
}

func (cfg Config) options(lx fx.Lifecycle) (mod module, err error) {
	mod.Ctx = ctxutil.WithLifecycle(context.Background(), lx) // libp2p lifecycle
	mod.Log = cfg.log.WithFields(log.F{
		"ns":   cfg.ns,
		"type": "client",
	})
	mod.Namespace = cfg.ns
	mod.Datastore = cfg.ds
	mod.Boot = cfg.d
	mod.Limit = cfg.queryLimit

	// options for host.Host
	mod.HostOpt = []config.Option{
		hostutil.MaybePrivate(cfg.psk),
		libp2p.Ping(false),
		libp2p.NoListenAddrs, // also disables relay
		libp2p.UserAgent("ww-client"),
	}

	// options for DHT
	mod.DHTOpt = []dht.Option{
		dht.Datastore(cfg.ds),
		dht.Mode(dht.ModeClient),
	}

	return
}

type module struct {
	fx.Out

	Ctx       context.Context
	Log       log.Logger
	Namespace string `name:"ns"`

	Datastore datastore.Batching
	Boot      discover.Strategy
	Limit     int           `name:"discover_limit"`
	Timeout   time.Duration `name:"discover_timeout"`

	HostOpt []config.Option
	DHTOpt  []dht.Option
}
