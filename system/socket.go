package system

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/rom"
	"golang.org/x/exp/slog"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
)

type Socket struct {
	NS  string
	ROM rom.ROM
	ID  routing.ID

	err atomic.Value

	// -----

	context  context.Context
	instance *wazergo.ModuleInstance[*Module]
	recv     <-chan segment
	mu       sync.Mutex
}

func (sock *Socket) String() string {
	// ww[12D3KooWQQfejFFG8UhrF8tGMvNF33y5s6j1T1pjxD9gLp7faBfG]
	return fmt.Sprintf("%s[%s]", sock.NS, sock.ROM)
}

func (sock *Socket) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", sock.NS, // the cluster
		"rom", sock.ROM, // the program being run
		"sock", sock.ID) // the thread that is executing
}

func (sock *Socket) Bind(ctx context.Context) (*Closer, error) {
	var closers *Closer

	// Spawn a new runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.
		NewRuntimeConfigCompiler().
		WithCloseOnContextDone(true))
	closers = closers.WithCloser(r)

	// Instantiate WASI.
	c, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		defer closers.Close(ctx)
		return nil, err
	}
	closers = closers.WithCloser(c)

	// // Instantiate Wetware host module.
	// sock, err := system.Instantiate(ctx, r)
	// if err != nil {
	// 	return err
	// }
	// defer sock.Close()

	// Compile guest module.
	compiled, err := r.CompileModule(sock.Ctx(), sock.ROM.Bytecode)
	if err != nil {
		defer closers.Close(ctx)
		return nil, err
	}
	closers = closers.WithCloser(compiled)

	// Instantiate the guest module, and configure host exports.

	// Now we instantiate the module.  Note that we use sock.Ctx() in
	// the call below.  It will contain some bound values that provide
	// an internal interface (github.com/stealthrocket/wazergo).
	mod, err := r.InstantiateModule(sock.Ctx(), compiled, wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithStartFunctions(). // don't automatically call _start while instanitating.
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		// WithArgs(sock.String()).
		WithEnv("ns", sock.NS).
		WithName(fmt.Sprintf("{%s %s, %s}", sock.NS, sock.ROM, sock.ID)).
		WithStdin(os.Stdin). // notice:  we connect stdio to host process' stdio
		WithStdout(os.Stdout).
		WithStderr(os.Stderr))
	if err != nil {
		defer closers.Close(ctx)
		return nil, err
	}
	closers = closers.WithCloser(mod)

	go sock.bind(mod)

	return closers, nil
}

func (sock *Socket) NewMessage() (transport.OutgoingMessage, error) {
	// TODO(someday):  we should write an Arena implementation that
	// uses msg.sock.instance to allocate segments directly to the
	// WASM process.  This would give us zero-copy message passing.
	//
	// It would look something like this:
	/*
		alloc := func(size int) []byte {
			return sock.alloc(size)
		}

		arena := capnp.NewMultiSegmentArenaWithAllocator(alloc)

		msg, seg, err := capnp.NewMessage(arena)
		// ...
	*/

	_, seg := capnp.NewMultiSegmentMessage(nil)
	message, err := rpccp.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	return &outgoing{
		message: message,
		sock:    sock,
	}, nil
}

func (sock *Socket) RecvMessage() (transport.IncomingMessage, error) {
	seg, err := sock.poll() // sock has reference to context
	if err != nil {
		return nil, err
	}

	buf, err := sock.deref(seg)
	if err != nil {
		return nil, err
	}

	msg, err := capnp.Unmarshal(buf)
	if err != nil {
		return nil, err
	}

	message, err := rpccp.ReadRootMessage(msg)
	return incoming(message), err
}

func (sock *Socket) bind(mod api.Module) {
	fn := mod.ExportedFunction("_start") // this is actually main()
	if fn == nil {
		sock.err.Store(errors.New("missing export: _start"))
		return
	}

	_, err := fn.Call(sock.Ctx())
	switch err.(*sys.ExitError).ExitCode() {
	case 0:
		return

	case sys.ExitCodeContextCanceled:
		sock.err.Store(context.Canceled)

	case sys.ExitCodeDeadlineExceeded:
		sock.err.Store(context.DeadlineExceeded)

	default:
		sock.err.Store(err)
	}
}

func (sock *Socket) Close() error {
	return sock.instance.Close(sock.context)
}

func (sock *Socket) Ctx() context.Context {
	return wazergo.WithModuleInstance[*Module](
		sock.context,
		sock.instance)
}

// poll returns the next segment sent to us by the guest.
// It may block, but will automatically unblock if/when
// sock.context expires.  The intent is for the host-side
// capnp transport to call this as part of its implementation
// of RecvMessage().
func (sock *Socket) poll() (segment, error) {
	select {
	case seg, ok := <-sock.recv:
		if ok {
			return seg, nil
		}
		return segment{}, rpc.ErrConnClosed

	case <-sock.context.Done():
		return segment{}, sock.context.Err()
	}
}

func (sock *Socket) alloc(size uint32) (segment, error) {
	sock.mu.Lock()
	defer sock.mu.Unlock()

	alloc := sock.instance.ExportedFunction("alloc")
	stack, err := alloc.Call(sock.context, api.EncodeU32(size))
	if err != nil {
		return segment{}, err
	}

	seg := segment{}.LoadValue(
		sock.instance.Memory(),
		stack)
	return seg, nil
}

func (sock *Socket) deref(seg segment) (types.Bytes, error) {
	sock.mu.Lock()
	defer sock.mu.Unlock()

	if b, ok := seg.LoadFrom(sock.instance.Memory()); ok {
		return b, nil
	}

	return nil, fmt.Errorf("%v: out of bounds", seg)
}

// notify calls the guest's exported __send function, which takes a
// segment (in the form of an (i32, i32) pair) and enqueues it onto
// the input buffer.  The guest's runtime will take it from there.
func (sock *Socket) notify(seg segment) error {
	sock.mu.Lock()
	defer sock.mu.Unlock()

	mem := sock.instance.Memory()
	handler := sock.instance.ExportedFunction("handle")

	// TODO:  sync.Pool
	// An over-engineered / possibly-too-clever optimization would be to
	// use []byte buffers of length 8 obtained from bufferpool.Default
	// instead of []uint64 slices of length 1.
	//
	// The default minimum allocation size for bufferpool.Default is 1Kb,
	// so on second thought, it's probably better to set up a dedicated
	// sync.Pool instance that serves fixed-size []uint64 instances.
	stack := make([]uint64, 1)
	seg.StoreValue(mem, stack)

	if err := handler.CallWithStack(sock.context, stack); err != nil {
		return err
	}

	status := api.DecodeI32(stack[0])
	if status == 0 {
		return nil
	}

	return types.Errno(status)
}

type incoming rpccp.Message

func (msg incoming) Message() rpccp.Message {
	return rpccp.Message(msg)
}

func (msg incoming) Release() {
	capnp.Struct(msg).Message().Release()
}

type outgoing struct {
	message rpccp.Message
	sock    *Socket
}

func (out outgoing) Message() rpccp.Message {
	return out.message
}

func (out outgoing) Release() {
	out.message.Release()
}

func (out outgoing) Send() error {
	msg, err := out.message.Message().Marshal()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	size := uint32(len(msg))
	seg, err := out.sock.alloc(size)
	if err != nil {
		return fmt.Errorf("alloc: %w", err)
	}

	buf, err := out.sock.deref(seg)
	if err != nil {
		return fmt.Errorf("deref: %w", err)
	}

	// copy buf into the process' linear memory.
	// There has *got* to be a way to avoid doing this...
	copy(buf, msg)
	return out.sock.notify(seg)
}
