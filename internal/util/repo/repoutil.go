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
	if path != "" {
		path = filepath.Join(path, "plugins")
	}

	// Load any external plugins if available on path
	plugins, err := loader.NewPluginLoader(path)
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
	/***********************************************************************************
	 *																				   *
	 *	Full documentation available at:											   *
	 *	https://github.com/ipfs/go-ipfs/blob/master/docs/config.md#table-of-contents   *
	 *																				   *
	 ***********************************************************************************/

	// Remove default bootstrap nodes.
	//
	// TODO:  provide facilities for users to specify bootstrap nodes.
	// 		  See:  https://github.com/ipfs/go-ipfs/blob/ce78064335c9923abb6540dc7a3cd512672a62bc/docs/experimental-features.md
	cfg.Bootstrap = nil

	// Default values are really huge (600 & 900, respectively).  Cut this down to
	// something more reasonable.
	//
	// TODO:  investigate whether this causes issues.  Check the following:
	//
	//			- Connection churn (frequent calls to `BasicConnMgr.trim` ?)
	// 			- Are pubsub conns protected?
	cfg.Swarm.ConnMgr.LowWater = 32
	cfg.Swarm.ConnMgr.HighWater = 128

	// Gateway seems to be disabled by default.  If this turns out to be wrong, try
	// uncommenting the lines below.
	/*
		cfg.Gateway = config.Gateway{}
		cfg.Addresses.API = nil
		cfg.Addresses.Gateway = nil
	*/
}
