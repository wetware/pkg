package ww

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"

	service "github.com/lthibault/service/pkg"
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
	if err := setupPlugins(path); err != nil {
		return nil, errors.Wrap(err, "setup plugins")
	}

	return fsrepo.Open(path)
}

func mkRepo(path string) (repo.Repo, error) {
	if err := setupPlugins(""); err != nil {
		return nil, errors.Wrap(err, "setup plugins")
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048) // TODO:  this should either be configurable or a const
	if err != nil {
		return nil, errors.Wrap(err, "new config")
	}

	// Create the repo with the config
	if err = fsrepo.Init(path, cfg); err != nil {
		return nil, errors.Wrap(err, "fsrepo init")
	}

	return fsrepo.Open(path)
}

// setupPlugins must be called before creating a repo in order to load
// preloaded (= built-in) plugins.
func setupPlugins(path string) error {
	// Load any external plugins if available on path
	plugins, err := loader.NewPluginLoader(filepath.Join(path, "plugins"))
	if err != nil {
		return errors.Wrap(err, "load plugins")
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return errors.Wrap(err, "init plugins")
	}

	if err := plugins.Inject(); err != nil {
		return errors.Wrap(err, "inject plugins")
	}

	return nil
}
