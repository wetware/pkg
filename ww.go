package ww

import (
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"io"
	"net"
	"runtime"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"golang.org/x/exp/slog"

	"github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/proto"
)

type SystemError interface {
	error
	ExitCode() uint32
}

type Viewport interface {
	View() view.View
}

// Ww is the execution context for WebAssembly (WASM) bytecode,
// allowing it to interact with (1) the local host and (2) the
// cluster environment.
type Ww struct {
	NS, Name string

	ID       routing.ID
	Host     local.Host
	Viewport Viewport

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// String returns the cluster namespace in which the wetware is
// executing. If ww.NS has been assigned a non-empty string, it
// returns the string unchanged.  Else, it defaults to "ww".
//
// This may change in the future.
func (ww *Ww) String() string {
	if ww.NS != "" {
		return ww.NS
	}

	return "ww"
}

func (ww Ww) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", ww.NS,
		"id", ww.ID,
		"peer", ww.Host.ID(),
		"hostname", ww.Name)
}

// Bind a system socket to Cap'n Proto RPC.  This method satisfies
// the system.Bindable interface.
func (ww *Ww) Bind(sock *system.Socket) *rpc.Conn {
	sock.Name = ww.Name + "." + ww.NS // subdomain, e.g. foo.ww
	sock.Net = ww.Host
	sock.View = cluster.View(ww.Viewport.View())

	// NOTE:  no auth is actually performed here.  The client doesn't
	// even need to pass a valid signer; the login call always succeeds.
	server := core.Terminal_NewServer(sock)
	client := capnp.NewClient(server)

	options := &rpc.Options{
		ErrorReporter:   system.ErrorReporter{Logger: sock.Logger},
		BootstrapClient: client,
	}

	return rpc.NewConn(rpc.NewStreamTransport(sock.Host), options)
}

// Exec compiles and runs the ww instance's ROM in a WASM runtime.
// It returns any error produced by the compilation or execution of
// the ROM.
func (ww *Ww) Exec(ctx context.Context, rom rom.ROM) error {
	// Spawn a new runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.
		NewRuntimeConfigCompiler().
		WithCloseOnContextDone(true))
	defer r.Close(ctx)

	// Instantiate WASI.
	c, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	// Instantiate wetware system socket.
	sock, err := wazergo.Instantiate(ctx, r, system.SocketModule,
		system.WithLogger(ww.Logger()),
		system.Bind(ww))
	if err != nil {
		return err
	}
	ctx = wazergo.WithModuleInstance(ctx, sock) // bind sock to context
	defer sock.Close(ctx)

	// Compile guest module.
	compiled, err := r.CompileModule(ctx, rom.Bytecode)
	if err != nil {
		return err
	}
	defer compiled.Close(ctx)

	// Instantiate the guest module, and configure host exports.
	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithStartFunctions(). // don't automatically call _start while instanitating.
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithArgs(rom.String()). // TODO(soon):  use content id
		WithEnv("ns", ww.String()).
		WithEnv("WW_DEBUG", "true").
		WithName(rom.String()).
		WithStdin(ww.Stdin). // notice:  we connect stdio to host process' stdio
		WithStdout(ww.Stdout).
		WithStderr(ww.Stderr))
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	return ww.run(ctx, mod)
}

func (ww Ww) run(ctx context.Context, mod api.Module) error {
	// Grab the the main() function and call it with the system context.
	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return errors.New("missing export: _start")
	}

	for {
		// TODO(performance):  fn.CallWithStack(ctx, nil)
		_, err := fn.Call(ctx)
		switch e := err.(type) {
		case net.Error:
			if e.Timeout() {
				sleep(ctx)
				continue
			}

		case SystemError:
			switch e.ExitCode() {
			case 0:
				return nil

			case sys.ExitCodeContextCanceled:
				return context.Canceled

			case sys.ExitCodeDeadlineExceeded:
				return context.DeadlineExceeded

			default:
				slog.Default().Debug(err.Error(),
					"version", proto.Version,
					"ns", ww.String(),
					"rom", mod.Name())
			}

		}

		return err
	}
}

func sleep(ctx context.Context) {
	select {
	case <-time.After(time.Millisecond):
	case <-ctx.Done():
	}
}
