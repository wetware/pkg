package ww

import (
	"context"
	"fmt"
	"time"

	log "github.com/lthibault/log/pkg"
	service "github.com/lthibault/service/pkg"
	"github.com/pkg/errors"

	"github.com/ipfs/go-ipfs/core"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/event"
)

/*
	runtime.go contains the wetware runtime
*/

// Runtime encapsulates global state for each host.
type Runtime struct {
	/********************
	*	static config	*
	*********************/
	ns, repoPath string
	ttl          time.Duration
	clientMode   bool

	buildCfg core.BuildCfg

	/********************
	*	runtime state	*
	*********************/
	log log.Logger
	ctx context.Context

	fs [256]*filter

	node *core.IpfsNode
	api  iface.CoreAPI
}

func (r *Runtime) setOptions(opt []Option) error {
	return applyOpt(r, withDefault(opt)...)
}

// Verify configuration.  Returns a descriptive error if a constraint is violated.
func (r *Runtime) Verify() (err error) {
	for _, validate := range []func() error{
		func() error { return assertNotNil(r.log, "logger must be set") },
		func() error { return assertNotEmpty(r.ns, "namespace must be specified") },
		validateBuildConfig(r),
	} {
		if err = validate(); err != nil {
			break
		}
	}
	return
}

// Bind a Host to the runtime.
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
		provideIPFS(r),
		provideCluster(r),

		/*************************************
		 * Inject dependencies into the Host *
		 *************************************/
		inject(r, h),

		/*******************************
		 * Start the Host's event loop *
		 *******************************/
		runEventLoop(r.ctx, h),
	}
}

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

// inject dependencies from a runtime into a host
func inject(r *Runtime, h *Host) service.Service {
	return service.Hook{
		OnStart: func() (err error) {
			r.log = r.log.WithFields(log.F{
				"id":    r.node.PeerHost.ID(),
				"addrs": r.node.PeerHost.Addrs(),
			})

			h.log = r.log
			h.host = r.node.PeerHost
			h.CoreAPI = r.api

			return
		},
	}
}

func runEventLoop(ctx context.Context, h *Host) service.Service {
	var sub event.Subscription
	return service.Hook{
		OnStart: func() (err error) {
			if sub, err = h.EventBus().Subscribe([]interface{}{
				new(EvtHeartbeat),
				// new(Event),
			}); err == nil {
				go h.loop(sub)
			}
			return
		},
		OnStop: func() error {
			return sub.Close()
		},
	}
}

/*
	Validation helpers
*/

func validateBuildConfig(r *Runtime) func() error {
	return func() (err error) {
		for _, verify := range []func() error{
			func() error {
				return assertNotNil(r.buildCfg, "build config must be set")
			},
			func() error {
				return assertTrue(r.buildCfg.Online, "networking must be enabled")
			},
			verifyClientMode(r),
		} {
			if err = verify(); err != nil {
				break
			}
		}

		return
	}
}

func verifyClientMode(r *Runtime) func() error {
	return func() (err error) {
		if !r.clientMode {
			return
		}

		for _, verify := range []func() error{
			func() error {
				return assertFalse(r.buildCfg.Permanent, "client nodes must be temporary")
			},
			func() error {
				return assertTrue(r.buildCfg.NilRepo, "client nodes must have a NilRepo")
			},
			func() error {
				// N.B.:  this is different than r.buildCfg.NilRepo.  If NilRepo == true
				//		  but r.buildCfg.Repo != nil, then the `Repo` value will end up
				//		  being used.
				return assertNil(r.buildCfg.Repo, "client nodes cannot declare a repo")
			},
			func() error {
				// N.B.:  this is an incomplete check.  We can't evaluate equality
				// 		  between function types in Go, so the best we can do is ensure
				// 		  that _something_ (hopefully `libp2p.DHTClientOption`) was set.
				return assertNotNil(r.buildCfg.Routing,
					"client nodes must use libp2p.DHTClientOption")
			},
		} {
			if err = verify(); err != nil {
				break
			}
		}

		return
	}
}

func assertFalse(b bool, msgAndArgs ...interface{}) error {
	return assertTrue(!b, "expected false, got true")
}

func assertTrue(b bool, msgAndArgs ...interface{}) error {
	if !b {
		return fmtErr("expected true, got false", msgAndArgs...)
	}
	return nil
}

func assertNil(obj interface{}, msgAndArgs ...interface{}) error {
	if obj != nil {
		return fmtErr(fmt.Sprintf("expected nil object, got %T", obj), msgAndArgs...)
	}
	return nil
}

func assertNotNil(obj interface{}, msgAndArgs ...interface{}) error {
	if obj == nil {
		return fmtErr("unexpected nil object", msgAndArgs...)
	}
	return nil
}

func assertNotEmpty(obj interface{}, msgAndArgs ...interface{}) error {
	switch v := obj.(type) {
	case string:
		if v == "" {
			return fmtErr("unexpected empty string", msgAndArgs...)
		}
	default:
		panic(obj)
	}

	return nil
}

func fmtErr(defaultMsg string, msgAndArgs ...interface{}) error {
	switch len(msgAndArgs) {
	case 0:
		return errors.New(defaultMsg)
	case 1:
		return errors.Errorf("%s", msgAndArgs[0])
	default:
		return errors.Errorf("%s", msgAndArgs[0], msgAndArgs[1:])
	}
}
