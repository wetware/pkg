// Package boot contains dependencies for injection via https://go.uber.org/fx
package boot

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"

	ww "github.com/lthibault/wetware/pkg"
)

// Env encapsulates runtime state
type Env interface {
	String(string) string
}

// Provide dependency
func Provide(env Env) fx.Option {
	return fx.Provide(
		newContext,
		newRepo(env),
		newIPFSNode,
		newHost,
	)
}

/*
	Constructors
*/

func newContext(lx fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}

func newRepo(env Env) func(context.Context) (repo.Repo, error) {
	return func(ctx context.Context) (_ repo.Repo, err error) {
		switch path := env.String("repo"); path {
		case "":
			if path, err = ioutil.TempDir("", "ww-*"); err != nil {
				return nil, errors.Wrap(err, "tempdir")
			}

			return mkOrLoadRepo(ctx, path)
		case "auto":
			if path, err = config.PathRoot(); err != nil { // default repo path from IPFS config
				return nil, err // shouldn't be possible
			}

			return loadRepo(path)
		default:
			return mkOrLoadRepo(ctx, path)
		}
	}
}

func newIPFSNode(ctx context.Context, repo repo.Repo) (*core.IpfsNode, error) {
	return core.NewNode(ctx, &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	})
}

func newHost(lx fx.Lifecycle, node *core.IpfsNode) (ww.Host, error) {
	h, err := ww.New(node)
	if err != nil {
		return nil, err
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return h.Close()
		},
	})

	return h, nil
}

/*
	helper functions
*/

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
