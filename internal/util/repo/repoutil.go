package repoutil

import (
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/pkg/errors"
)

// DefaultKeySize for new repositories.  Can be overridden using `WithKeySize`.
const DefaultKeySize = 2048

// InitRepo creates a new filesystem-backed repository.
func InitRepo(path string, opt ...Option) (err error) {
	if err := SetupPlugins(""); err != nil {
		return errors.Wrap(err, "setup plugins")
	}

	spec := specWithOptions(opt)
	if spec.Config == nil {
		if spec.Config, err = config.Init(spec.Printer, spec.KeySize); err != nil {
			return errors.Wrap(err, "create config")
		}

		setConfig(spec.Config)
	}

	// Create the repo with the config
	return errors.Wrap(fsrepo.Init(path, spec.Config), "create repo")
}

// SetupPlugins must be called before loading a repo in order to load preloaded
// (= built-in) plugins.
func SetupPlugins(path string) error {
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

func setConfig(cfg *config.Config) {
	// // https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#ipfs-filestore
	// cfg.Experimental.FilestoreEnabled = true
	// // https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#ipfs-urlstore
	// cfg.Experimental.UrlstoreEnabled = true
	// // https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#directory-sharding--hamt
	// cfg.Experimental.ShardingEnabled = true
	// // https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#ipfs-p2p
	// cfg.Experimental.Libp2pStreamMounting = true
	// // https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#p2p-http-proxy
	// cfg.Experimental.P2pHttpProxy = true
	// // https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#quic
	// cfg.Experimental.QUIC = true
	// // https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#strategic-providing
	// cfg.Experimental.StrategicProviding = true
}
