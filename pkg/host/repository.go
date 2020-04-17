package host

import (
	"context"
	"io/ioutil"
	"os"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	repoutil "github.com/lthibault/wetware/internal/util/repo"
	"github.com/pkg/errors"
)

func newRepository(ctx context.Context, cfg *Config) (_ repo.Repo, err error) {
	switch path := cfg.repoPath; path {
	case "":
		if path, err = ioutil.TempDir("", "ww-*"); err != nil {
			return nil, errors.Wrap(err, "tempdir")
		}

		cfg.log.WithField("path", path).Debug("creating temporary repo")
		return mkRepo(path)
	case "auto":
		// Use default repo path from IPFS config
		if path, err = config.PathRoot(); err != nil {
			return nil, err // shouldn't be possible
		}

		cfg.log.WithField("path", path).Debug("using default IPFS path")
		return loadRepo(path)
	default:
		if err := os.MkdirAll(path, 0770); os.IsExist(err) {
			cfg.log.WithField("path", path).Debug("loading repo")
			return loadRepo(path)
		}

		cfg.log.WithField("path", path).Debug("creating repo")
		return mkRepo(path)
	}
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
