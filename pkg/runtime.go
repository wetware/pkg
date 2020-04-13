package ww

import (
	"context"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/repo"
	log "github.com/lthibault/log/pkg"
	service "github.com/lthibault/service/pkg"
	"github.com/pkg/errors"
)

/*
	runtime.go contains the wetware runtime
*/

// Runtime encapsulates global state for each host.
type Runtime struct {
	/********************
	*	static config	*
	*********************/
	repoPath string

	/********************
	*	runtime state	*
	*********************/
	log log.Logger
	ctx context.Context

	repo repo.Repo
	node *core.IpfsNode
}

func (r *Runtime) setOptions(opt []Option) (err error) {
	for _, f := range withDefault(opt) {
		if err = f(r); err != nil {
			break
		}
	}

	return
}

// Verify configuration.  Returns a descriptive error if a constraint is violated.
func (r *Runtime) Verify() error {
	for _, v := range []struct {
		test func(*Runtime) bool
		emsg string
	}{{
		test: func(r *Runtime) bool { return r.log != nil },
		emsg: "logger must not be nil",
	}} {
		if !v.test(r) {
			return errors.New(v.emsg)
		}
	}
	return nil
}

// Bind performs dependency injection on the Host and sets the host's root service.
func (r *Runtime) Bind(h *Host) {
	/*
	 *	A host's root service is a tree of (start, stop) functions that are called
	 *	recursively.  Calling h.root.Start/Stop effectively starts/stops the Host.
	 */
	h.root = service.Array{
		/***************************************
		 * Provide dependencies to the Runtime *
		 ***************************************/
		provideContext(r),
		provideRepo(r),
		provideIPFSNode(r),
		provideStreamHandlers(r),

		/********************************
		 * Manage background goroutines *
		 ********************************/
		//  runHeartbeat(),

		/*************************************
		 * Inject dependencies into the Host *
		 *************************************/
		inject(r, h), // inject dependencies & start background processes
	}
}

func inject(r *Runtime, h *Host) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			r.log.Debug("starting host")

			// inject dependencies into host
			h.log = r.log
			h.host = r.node.PeerHost

			if h.CoreAPI, err = coreapi.NewCoreAPI(r.node); err != nil {
				return
			}

			return
		},
		OnStop: func() (err error) {
			r.log.Debug("shuting down host")
			return
		},
	}
}
