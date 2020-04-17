package host

import (
	"context"

	"go.uber.org/fx"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/repo"
	iface "github.com/ipfs/interface-go-ipfs-core"
)

type hostParam struct {
	fx.In

	Cfg  *Config
	Node *core.IpfsNode
	API  iface.CoreAPI
}

func module(h *Host, opt []Option) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(opt),
		fx.Provide(
			newCtx,
			newConfig,
			newRepository,
			newBuildCfg,
			core.NewNode,       // TODO:  check if Fx chokes on variadic args
			coreapi.NewCoreAPI, // TODO:  check if Fx chokes on variadic args
			newHost,
		),
		fx.Populate(h),
	)
}

func newHost(ctx context.Context, lx fx.Lifecycle, p hostParam) Host {
	for _, hook := range []fx.Hook{
		eventloop(ctx, p.Node.PeerHost),
		announce(ctx, p.Cfg, p.API.PubSub(), p.Node.PeerHost),
	} {
		lx.Append(hook)
	}

	return Host{
		log:  p.Cfg.Log(),
		host: p.Node.PeerHost,
		api:  p.API,
	}
}

func newCtx(lx fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}

func newBuildCfg(repo repo.Repo) (*core.BuildCfg, error) {
	return &core.BuildCfg{
		Online:    true,
		Permanent: true,
		Routing:   libp2p.DHTOption,
		ExtraOpts: map[string]bool{
			"pubsub": true,
			// "ipnsps": false,
			// "mplex":  false,
		},
		Repo: repo,
	}, nil
}
