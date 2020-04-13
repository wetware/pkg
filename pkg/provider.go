package ww

import (
	"context"
	"io/ioutil"
	"os"

	service "github.com/lthibault/service/pkg"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"

	repoutil "github.com/lthibault/wetware/internal/util/repo"
)

func provideContext(r *Runtime) service.Service {
	ctx, cancel := context.WithCancel(context.Background())
	return service.Hook{
		OnStart: func() error {
			r.ctx = ctx
			return nil
		},
		OnStop: func() error {
			cancel()
			return nil
		},
	}
}

func provideRepo(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			r.repo, err = newRepo(r.ctx, r.repoPath)
			return
		},
	}
}

func provideIPFSNode(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			r.log.Trace("starting IPFS node")

			r.node, err = core.NewNode(r.ctx, &core.BuildCfg{
				Online:  true,
				Routing: libp2p.DHTOption,
				Repo:    r.repo,
			})
			return
		},
		OnStop: func() error {
			r.log.Trace("stopping IPFS node")
			return r.node.Close()
		},
	}
}

func provideStreamHandlers(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() error {
			r.log.Trace("registering stream handlers")

			r.node.PeerHost.SetStreamHandler("test", func(s network.Stream) {
				r.log.
					WithField("proto", "test").
					WithField("stat", s.Stat()).
					Info("stream handled")

				s.Reset()
			})

			return nil
		},
	}
}

/*
	helper functions
*/

func newRepo(ctx context.Context, path string) (repo.Repo, error) {
	switch path {
	case "":
		path, err := ioutil.TempDir("", "ww-*")
		if err != nil {
			return nil, errors.Wrap(err, "tempdir")
		}

		return mkOrLoadRepo(ctx, path)
	case "auto":
		path, err := config.PathRoot() // default repo path from IPFS config
		if err != nil {
			return nil, err // shouldn't be possible
		}

		return loadRepo(path)
	default:
		return mkOrLoadRepo(ctx, path)
	}
}

func mkOrLoadRepo(ctx context.Context, path string) (repo.Repo, error) {
	if err := os.MkdirAll(path, 0770); os.IsExist(err) {
		return loadRepo(path)
	}

	return mkRepo(path)
}

func loadRepo(path string) (repo.Repo, error) {
	if err := repoutil.SetupPlugins(path); err != nil {
		return nil, errors.Wrap(err, "setup plugins")
	}

	return fsrepo.Open(path)
}

func mkRepo(path string) (repo.Repo, error) {
	// SetupPlugins is called by InitRepo, so no need to call it again.
	if err := repoutil.InitRepo(path); err != nil {
		return nil, errors.Wrap(err, "init")
	}

	return fsrepo.Open(path)
}
