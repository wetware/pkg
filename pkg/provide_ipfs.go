package ww

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"os"

	log "github.com/lthibault/log/pkg"
	service "github.com/lthibault/service/pkg"
	"github.com/pkg/errors"

	"github.com/ipfs/go-datastore"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"

	repoutil "github.com/lthibault/wetware/internal/util/repo"
)

func provideIPFS(r *Runtime) service.Service {
	return service.Array{
		provideIPFSNode(r),
		provideCoreAPI(r),
	}
}

func provideIPFSNode(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() error {
			cfg, err := newBuildCfg(r)
			if err != nil {
				return errors.Wrap(err, "build config")
			}

			if r.node, err = core.NewNode(r.ctx, cfg); err != nil {
				return errors.Wrap(err, "create node")
			}

			return nil
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
	build config helper functions
*/

func newBuildCfg(r *Runtime) (*core.BuildCfg, error) {
	repo, err := newRepo(r.ctx, r.log, r.repoPath)
	if err != nil {
		return nil, err
	}

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

func newClientRepo(log log.Logger) (repo.Repo, error) {
	var c config.Config
	priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, err
	}

	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}

	privkeyb, err := priv.Bytes()
	if err != nil {
		return nil, err
	}

	c.Identity.PeerID = pid.Pretty()
	c.Identity.PrivKey = base64.StdEncoding.EncodeToString(privkeyb)

	// TODO:  we don't want peers to try to connect with a client.  Client isn't listening...
	// c.Discovery.MDNS.Enabled = true
	// c.Discovery.MDNS.Interval = "10s"

	c.Routing.Type = "dht"

	return &repo.Mock{
		D: datastore.NewNullDatastore(),
		C: c,
	}, nil
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
