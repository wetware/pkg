//go:generate mockgen -source=host.go -destination=../../internal/mock/pkg/host/host.go -package=mock_host

package host

import (
	"context"
	"errors"
	"fmt"
	"time"

	"capnproto.org/go/capnp/v3"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/debug"
	anchor_api "github.com/wetware/ww/internal/api/anchor"
	api "github.com/wetware/ww/internal/api/cluster"
	process_api "github.com/wetware/ww/internal/api/process"
	pubsub_api "github.com/wetware/ww/internal/api/pubsub"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/pkg/csp"
	"github.com/wetware/ww/pkg/pubsub"
)

var Capability = casm.BasicCap{
	"host/packed",
	"host"}

/*----------------------------*
|                             |
|    Client Implementations   |
|                             |
*-----------------------------*/

type Host api.Host

func (h Host) AddRef() Host {
	return Host(capnp.Client(h).AddRef())
}

func (h Host) Release() {
	capnp.Client(h).Release()
}

func (h Host) View(ctx context.Context) (cluster.View, capnp.ReleaseFunc) {
	f, release := api.Host(h).View(ctx, nil)
	return cluster.View(f.View()), release
}

func (h Host) PubSub(ctx context.Context) (pubsub.Router, capnp.ReleaseFunc) {
	f, release := api.Host(h).PubSub(ctx, nil)
	return pubsub.Router(f.PubSub()), release
}

func (h Host) Root(ctx context.Context) (anchor.Anchor, capnp.ReleaseFunc) {
	f, release := api.Host(h).Root(ctx, nil)
	return anchor.Anchor(f.Root()), release
}

func (h Host) Debug(ctx context.Context) (debug.Debugger, capnp.ReleaseFunc) {
	f, release := api.Host(h).Debug(ctx, nil)
	return debug.Debugger(f.Debugger()), release
}

func (h Host) Executor(ctx context.Context) (csp.Executor, capnp.ReleaseFunc) {
	f, release := api.Host(h).Executor(ctx, nil)
	return csp.Executor(f.Executor()), release
}

/*---------------------------*
|                            |
|    Server Implementation   |
|                            |
*----------------------------*/

type ViewProvider interface {
	View() cluster.View
}

type PubSubProvider interface {
	PubSub() pubsub.Router
}

type AnchorProvider interface {
	Anchor() anchor.Anchor
}

type DebugProvider interface {
	Debugger() debug.Debugger
}

type ExecutorProvider interface {
	Executor(host api.Host) csp.Executor
}

// Server provides the Host capability.
type Server struct {
	ViewProvider     ViewProvider
	PubSubProvider   PubSubProvider
	AnchorProvider   AnchorProvider
	DebugProvider    DebugProvider
	ExecutorProvider ExecutorProvider
}

func (s Server) Client() capnp.Client {
	return capnp.Client(s.Host())
}

func (s Server) Host() Host {
	return Host(api.Host_ServerToClient(s))
}

func (s Server) View(_ context.Context, call api.Host_view) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetView(capnp.Client(s.ViewProvider.View()))
	}

	return err
}

func (s Server) PubSub(_ context.Context, call api.Host_pubSub) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetPubSub(pubsub_api.Router(s.PubSubProvider.PubSub()))
	}

	return err
}

func (s Server) Root(_ context.Context, call api.Host_root) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetRoot(anchor_api.Anchor(s.AnchorProvider.Anchor()))
	}

	return err
}

func (s Server) Debug(_ context.Context, call api.Host_debug) error {
	res, err := call.AllocResults()
	if err == nil {
		debugger := s.DebugProvider.Debugger()
		err = res.SetDebugger(capnp.Client(debugger))
	}

	return err
}

func (s Server) Executor(ctx context.Context, call api.Host_executor) error {
	// TODO mikel do we need call.Go? The capability will be passed down, so I'd assume yes
	call.Go()
	host := call.Args().Host()
	res, err := call.AllocResults()
	if err == nil {
		e := s.ExecutorProvider.Executor(host)
		err = res.SetExecutor(process_api.Executor(e))
	}

	if err != nil {
		return err
	}

	// TODO mikel
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-time.After(1 * time.Minute):
		err = errors.New("executor timeout")
	}

	return err
}

func (s Server) SayHi(_ context.Context, call api.Host_sayHi) error {
	fmt.Println("Hi!") // TODO mikel the objective to see this appear in the terminal
	return nil
}
