package iostream_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/ocap/iostream"
)

func TestStream(t *testing.T) {
	t.Parallel()

	buf := &mockCloser{Buffer: new(bytes.Buffer)}
	stream := iostream.New(buf, nil)
	defer stream.Release()

	stream.AddRef().Release() // for test coverage

	f, release := stream.WriteString(context.TODO(), "hello, world!")
	defer release()

	require.NoError(t, f.Err(), "should write")
	assert.Equal(t, "hello, world!", buf.String())

	err := stream.Close(context.TODO())
	require.NoError(t, err, "should close")

	buf.Reset()
	f, release = stream.WriteString(context.TODO(), "should fail")
	defer release()

	require.ErrorIs(t, f.Err(), iostream.ErrClosed,
		"write to closed stream should fail")
	assert.Zero(t, buf.Len(),
		"data should not be written to closed stream")
	assert.True(t, buf.Closed,
		"writer should have been closed")
}

func TestProvider(t *testing.T) {
	rbuf := &mockCloser{Buffer: bytes.NewBufferString("hello, world!")}
	p := iostream.NewProvider(rbuf, nil)
	defer p.Release()

	p.AddRef().Release() // for test coverage

	buf := &mockCloser{Buffer: new(bytes.Buffer)}
	f, release := p.Provide(context.TODO(), iostream.New(buf, nil))
	defer release()

	require.NoError(t, f.Err(), "should succeed")
	require.Equal(t, "hello, world!", buf.String())
	assert.True(t, buf.Closed, "writer should have been closed")

	p.Release() // signal that the reader should be closed
	assert.True(t, rbuf.Closed, "reader should have been closed")
}

type mockCloser struct {
	Closed bool
	*bytes.Buffer
}

func (mc *mockCloser) Close() error {
	mc.Closed = true
	return nil
}
