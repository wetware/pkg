// Package boot contains dependencies for injection via https://go.uber.org/fx
package boot

import (
	"context"
	"io/ioutil"
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
	// GlobalString(string) string
	// ...
}

// Provide dependency
func Provide(Env) fx.Option {
	return fx.Provide(
		newContext,
		newRepo,
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

func newRepo(ctx context.Context) (repo.Repo, error) {
	// setupPlugins must be called before creating a repo in order to load "preloaded"
	// (= built-in) plugins.
	if err := setupPlugins(""); err != nil {
		return nil, err
	}

	repoPath, err := createTempRepo(ctx)
	if err != nil {
		return nil, err
	}

	return fsrepo.Open(repoPath)
}

func newIPFSNode(ctx context.Context, repo repo.Repo) (*core.IpfsNode, error) {
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	return core.NewNode(ctx, nodeOptions)
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

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return errors.Wrap(err, "error loading plugins")
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return errors.Wrap(err, "error initializing plugins")
	}

	if err := plugins.Inject(); err != nil {
		return errors.Wrap(err, "error initializing plugins")
	}

	return nil
}

func createTempRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return "", errors.Wrap(err, "failed to get temp dir")
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to init ephemeral node")
	}

	return repoPath, nil
}
