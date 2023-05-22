package system

import (
	"context"
	"encoding/binary"
	"errors"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/exp/bufferpool"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	i32     = api.ValueTypeI32
	bufSize = 1024 * 16 // KB
)

// HostModule is a bidirectional stream between a host and its guest.
type HostModule struct {
	api.Closer
}

// Instantiate a "system connection":  a bidirectional byte-stream between
// a guest and its host.
func Instantiate(ctx context.Context, r wazero.Runtime) (*HostModule, error) {
	c, err := buildModule(ctx, r)
	if err != nil {
		return nil, err
	}

	return &HostModule{
		Closer: c,
	}, nil
}

func (h HostModule) Close(ctx context.Context) error {
	return h.Closer.Close(ctx)
}

// Bind the wazero module to the host module, forming a bidirectional connection.
func (h *HostModule) Bind(ctx context.Context, mod api.Module) (*Conn, error) {
	conn, err := newConn(ctx, mod, bufSize, bufSize)
	if err == nil {
		go func() {
			t := NewTransport(ctx, conn)

			c := rpc.NewConn(t, &rpc.Options{
				// DEBUG
				BootstrapClient: capnp.ErrorClient(errors.New("Hello, Wetware!")),
				// --DEBUG
			})
			defer c.Close()

			select {
			case <-c.Done():
			case <-ctx.Done():
			}
		}()
	}

	return conn, nil
}

type Conn struct {
	sendPtr, recvPtr uint32
	sendCh, recvCh   chan []byte
	send             api.Function
}

func newConn(ctx context.Context, mod api.Module, hostBufSize, guestBufSize uint32) (*Conn, error) {
	init := mod.ExportedFunction("__init")
	if init != nil {
		return nil, errors.New("missing export: __init")
	}

	send := mod.ExportedFunction("__send")
	if init != nil {
		return nil, errors.New("missing export: __send")
	}

	conn := &Conn{
		sendCh: make(chan []byte, 16),
		recvCh: make(chan []byte, 16),
		send:   send,
	}
	ctx = WithConn(ctx, conn) // we have re-entrant calls
	conn.initBuffers(ctx, init)
	conn.sendLoop(ctx, mod.Memory())

	return conn, nil
}

// WithConn binds the system conn to a new context.  The returned context
// is a child of ctx.
func WithConn(ctx context.Context, conn *Conn) context.Context {
	return context.WithValue(ctx, keyConn{}, conn)
}

type keyConn struct{}

func (c *Conn) initBuffers(ctx context.Context, init api.Function) error {
	res, err := init.Call(ctx,
		api.EncodeU32(1024),
		api.EncodeU32(1024))
	if err == nil {
		c.sendPtr = api.DecodeU32(res[0])
		c.recvPtr = api.DecodeU32(res[1])
	}

	return err
}

func buildModule(ctx context.Context, r wazero.Runtime) (api.Closer, error) {
	return r.NewHostModuleBuilder("ww").

		// Init()
		NewFunctionBuilder().
		WithParameterNames("send_ptr", "recv_ptr").
		WithGoModuleFunction(initBuffers(),
			[]api.ValueType{i32, i32}, // params
			[]api.ValueType{}).        // results
		Export("__init_buffers").

		// Send()
		NewFunctionBuilder().
		WithParameterNames("recv_pos").
		WithGoModuleFunction(send(),
			[]api.ValueType{i32}, // params
			[]api.ValueType{}).   // results
		Export("__send").

		// link it all up...
		Instantiate(ctx)
}

func initBuffers() api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		conn := ctx.Value(keyConn{}).(*Conn)
		conn.sendPtr, conn.recvPtr = api.DecodeU32(stack[0]), api.DecodeU32(stack[1])
	}
}

func send() api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		conn := ctx.Value(keyConn{}).(*Conn)
		conn.hostSend(ctx, mod.Memory(), uint32(stack[0]))
	}
}

func (c *Conn) hostSend(ctx context.Context, mem api.Memory, recvPos uint32) {
	buf := c.recvBuffer(mem, recvPos)

	var fr frame
	for more := fr.Read(buf); more; more = fr.Read(buf) {
		select {
		case c.recvCh <- fr.Body:
			buf = buf[fr.Len():] // advance by n bytes

		case <-ctx.Done():
			return
		}
	}
}

func (c *Conn) recvBuffer(mem api.Memory, pos uint32) []byte {
	return buffer(mem, c.recvPtr, pos)
}

func (c *Conn) sendBuffer(mem api.Memory, pos uint32) []byte {
	return buffer(mem, c.sendPtr, pos)
}

func buffer(mem api.Memory, ptr, pos uint32) []byte {
	if buf, ok := mem.Read(ptr, pos); ok {
		return buf
	}

	panic("out of bounds")
}

type frame struct {
	header
	Body []byte
}

func (f *frame) Read(buf []byte) (more bool) {
	if len(buf) == 0 {
		return false
	}

	f.Body = bufferpool.Default.Get(f.header.Len())
	copy(f.Body, readHeader(f.header[:], buf))
	return true
}

type header [4]byte

func (h header) Len() int {
	return int(binary.LittleEndian.Uint32(h[0:4]))
}

// return a 4-byte length header
func readHeader(hdr, stream []byte) []byte {
	copy(hdr, stream[0:4])
	return stream[4:]
}

func (c *Conn) sendLoop(ctx context.Context, mem api.Memory) {
	ctx = WithConn(ctx, c)
	var hdr header

	for b := range c.sendCh {
		buf := c.sendBuffer(mem, c.sendPtr)

		// Encode length header
		binary.LittleEndian.PutUint32(hdr[:], uint32(len(b)))
		copy(buf[0:], hdr[:4])
		copy(buf[4:], b) // FIXME:  no bounds checking; can send len(b) > bufSize!

		// TODO: YOU ARE HERE
		//
		// Change the send functions to deal in uint64s that encode (offset, size)
		// tuples.  Then, create a `__alloc` guest export.  From there, you simply
		// need three ops for a send: (1) alloc buffer (2) copy to buffer, and (3)
		// add (offset, size) tuple to ring-buffer.
		//
		// WARNING:  Can we even do this?  What happens if we call `make([]byte)`
		// from multiple host threads?

		// Send frame data to guest.
		_, err := c.send.Call(ctx, uint64(4+len(b)))
		if err != nil {
			panic(err) // FIXME:  guest can cause host to panic(?)
		}

		bufferpool.Default.Put(b)
	}
}
