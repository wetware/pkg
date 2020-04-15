package ww

import (
	"context"
	"io/ioutil"
	"os"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	log "github.com/lthibault/log/pkg"
	service "github.com/lthibault/service/pkg"
	repoutil "github.com/lthibault/wetware/internal/util/repo"
	"github.com/pkg/errors"
)

func provideIPFS(r *Runtime) service.Service {
	return service.Array{
		provideRepo(r),
		provideIPFSNode(r),
		provideCoreAPI(r),
	}
}

func provideRepo(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			// Repo may have been set by passing a non-nil core.BuildCfg to the
			// withBuildConfig option.
			if !r.buildCfg.NilRepo && r.buildCfg.Repo == nil {
				r.buildCfg.Repo, err = newRepo(r.ctx, r.log, r.repoPath)
			}
			return
		},
	}
}

func provideIPFSNode(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			r.node, err = core.NewNode(r.ctx, &r.buildCfg)
			return
		},
		OnStop: func() error {
			return r.node.Close()
		},
	}
}

func provideCoreAPI(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			r.api, err = coreapi.NewCoreAPI(r.node)
			return
		},
	}
}

/*
	repo helper functions
*/

func newRepo(ctx context.Context, log log.Logger, path string) (_ repo.Repo, err error) {
	switch path {
	case "":
		if path, err = ioutil.TempDir("", "ww-*"); err != nil {
			return nil, errors.Wrap(err, "tempdir")
		}

		log.WithField("path", path).Debug("creating temporary repo")
		return mkRepo(path)
	case "auto":
		// Use default repo path from IPFS config
		if path, err = config.PathRoot(); err != nil {
			return nil, err // shouldn't be possible
		}

		log.WithField("path", path).Debug("using default IPFS path")
		return loadRepo(path)
	default:
		if err := os.MkdirAll(path, 0770); os.IsExist(err) {
			log.WithField("path", path).Debug("loading repo")
			return loadRepo(path)
		}

		log.WithField("path", path).Debug("creating repo")
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
