package ww

import (
	"context"
	"io/ioutil"
	"os"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
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
			r.repo, err = newRepo(r.ctx, r.repoPath)
			return
		},
	}
}

func provideIPFSNode(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			r.node, err = core.NewNode(r.ctx, &core.BuildCfg{
				Online:    true,
				Routing:   libp2p.DHTOption,
				Permanent: !r.tempNode,
				Repo:      r.repo,
				ExtraOpts: map[string]bool{
					"pubsub": true,
					// "ipnsps": false,
					// "mplex":  false,
				},
			})
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
