package system

import (
	"context"
	"fmt"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero/api"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
)

type Socket struct {
	context  context.Context
	instance *wazergo.ModuleInstance[*Module]
	recv     <-chan segment
}

func (sock *Socket) Close() error {
	return sock.instance.Close(sock.context)
}

func (sock *Socket) Ctx() context.Context {
	return wazergo.WithModuleInstance[*Module](
		sock.context,
		sock.instance)
}

func (sock *Socket) RPCTransport() rpc.Transport {
	return capnpTransport{sock: sock}
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
	if b, ok := seg.LoadFrom(sock.instance.Memory()); ok {
		return b, nil
	}

	return nil, fmt.Errorf("%v: out of bounds", seg)
}

// notify calls the guest's exported __send function, which takes a
// segment (in the form of an (i32, i32) pair) and enqueues it onto
// the input buffer.  The guest's runtime will take it from there.
func (sock *Socket) notify(seg segment) error {
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

type capnpTransport struct {
	sock *Socket
}

func (t capnpTransport) Close() error {
	return t.sock.Close()
}

func (t capnpTransport) NewMessage() (transport.OutgoingMessage, error) {
	// TODO(someday):  we should write an Arena implementation that
	// uses msg.sock.instance to allocate segments directly to the
	// WASM process.  This would give us zero-copy message passing.
	//
	// It would look something like this:
	/*
		alloc := func(size int) []byte {
			return t.sock.alloc(size)
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
		message:   message,
		transport: t,
	}, nil
}

func (t capnpTransport) RecvMessage() (transport.IncomingMessage, error) {
	seg, err := t.sock.poll() // sock has reference to context
	if err != nil {
		return nil, err
	}

	buf, err := t.sock.deref(seg)
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

func (t capnpTransport) sendMsg(msg []byte) error {
	size := uint32(len(msg))
	seg, err := t.sock.alloc(size)
	if err != nil {
		return fmt.Errorf("alloc: %w", err)
	}

	buf, err := t.sock.deref(seg)
	if err != nil {
		return fmt.Errorf("deref: %w", err)
	}

	// copy buf into the process' linear memory.
	// There has *got* to be a way to avoid doing this...
	copy(buf, msg)
	return t.sock.notify(seg)
}

type incoming rpccp.Message

func (msg incoming) Message() rpccp.Message {
	return rpccp.Message(msg)
}

func (msg incoming) Release() {
	capnp.Struct(msg).Message().Release()
}

type outgoing struct {
	message   rpccp.Message
	transport capnpTransport
}

func (msg outgoing) Message() rpccp.Message {
	return msg.message
}

func (msg outgoing) Release() {
	msg.message.Release()
}

func (msg outgoing) Send() error {
	buf, err := msg.message.Message().Marshal()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return msg.transport.sendMsg(buf)
}
