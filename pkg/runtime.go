package ww

import (
	"context"
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

	/********************
	*	runtime state	*
	*********************/
	log log.Logger
	ctx context.Context

	fs [256]*filter

	node *core.IpfsNode
	api  iface.CoreAPI
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
func (r *Runtime) Verify() (err error) {
	for _, validate := range []func() error{
		func() error { return assertNotNil(r.log, "logger must be set") },
		func() error { return assertNotEmpty(r.ns, "namespace must be specified") },
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
