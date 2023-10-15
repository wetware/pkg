package system

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
)

func TestSegment(t *testing.T) {
	t.Parallel()

	seg := segment(18 << 32) // offset=18
	seg |= 12                // size=12
	assert.Equal(t, uint32(18), seg.Offset(), "offset should be 18")
	assert.Equal(t, uint32(12), seg.Size(), "size should be 12")
}

func TestRead(t *testing.T) {
	t.Parallel()

	seg := segment(18 << 32) // offset=18
	seg |= 12                // size=12

	method := wazerotest.NewFunction(func(ctx context.Context, mod api.Module, sg uint64, d uint64) uint64 {
		offset := segment(sg).Offset()
		size := segment(sg).Size()
		require.Equal(t, seg.Offset(), offset, "offsets should match")
		require.Equal(t, seg.Size(), size, "sizes should match")

		return uint64(size) << 32
	})

	mem := wazerotest.NewFixedMemory(512)
	mod := wazerotest.NewModule(mem, method)

	stack := []uint64{
		uint64(seg),              // segment[offset=18, size=12]
		uint64(time.Millisecond), // timeout
	}

	buf := bytes.NewBufferString("hello, wasm!")
	sockRead{conn{buf}}.Call(context.Background(), mod, stack)
	e := effect(stack[0])
	require.NoError(t, e.Err(), "effect should not return error")
	require.Equal(t, seg.Size(), e.Bytes(),
		"should consume exactly %d bytes", seg.Size())
}

func TestWrite(t *testing.T) {
	t.Parallel()

	seg := segment(18 << 32) // offset=18
	seg |= 12                // size=12

	method := wazerotest.NewFunction(func(ctx context.Context, mod api.Module, sg uint64, d uint64) uint64 {
		offset := segment(sg).Offset()
		size := segment(sg).Size()
		require.Equal(t, seg.Offset(), offset, "offsets should match")
		require.Equal(t, seg.Size(), size, "sizes should match")

		return uint64(size) << 32
	})

	mem := wazerotest.NewFixedMemory(512)
	mod := wazerotest.NewModule(mem, method)

	stack := []uint64{
		uint64(seg),              // segment[offset=18, size=12]
		uint64(time.Millisecond), // timeout
	}

	buf := new(bytes.Buffer)
	sockWrite{conn{buf}}.Call(context.Background(), mod, stack)
	e := effect(stack[0])
	require.NoError(t, e.Err(), "effect should not return error")
	require.Equal(t, seg.Size(), e.Bytes(),
		"should consume exactly %d bytes", seg.Size())
}

func TestClose(t *testing.T) {
	t.Parallel()

	c := new(mockCloser)
	sockClose{c}.Call(context.Background(), nil, nil)
	assert.True(t, c.closed)
}

type mockCloser struct {
	net.Conn
	closed bool
}

func (c *mockCloser) Close() error {
	if c.closed {
		return errors.New("already called")
	}
	c.closed = true
	return nil
}

type conn struct{ *bytes.Buffer }

func (conn) Close() error                       { return nil }
func (conn) LocalAddr() net.Addr                { return nil }
func (conn) RemoteAddr() net.Addr               { return nil }
func (conn) SetDeadline(t time.Time) error      { return nil }
func (conn) SetReadDeadline(t time.Time) error  { return nil }
func (conn) SetWriteDeadline(t time.Time) error { return nil }

// func TestReadable(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()

// 	mem := wazerotest.NewFixedMemory(1024)
// 	mod := wazerotest.NewModule(mem)
// 	defer mod.Close(ctx)

// 	seg := segment(18 << 32) // offset=18
// 	seg |= 12                // size=12

// }
