package iostream_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/iostream"
)

func TestStream(t *testing.T) {
	t.Parallel()

	buf := &mockCloser{Buffer: new(bytes.Buffer)}
	stream := iostream.New(buf)
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
	const bufferSize = 2048
	const s1, s2 = "hello, world!\n", "hello again, world!\n"
	rc, wc := io.Pipe() // Client Reader/Writer
	rs, ws := io.Pipe() // Server Reader/Writer, could be replaced with bytes.Buffer

	defer rc.Close()
	defer wc.Close()
	defer rs.Close()
	defer ws.Close()

	// Write s1 to the client writer
	go func() {
		n, err := wc.Write([]byte(s1))
		require.NoError(t, err)
		assert.Equal(t, n, len(s1))
	}()

	// Provide the client reader to the server writer
	go func() {
		p := iostream.NewProvider(rc)
		defer p.Release()
		p.AddRef().Release() // For test coverage
		_, release := p.Provide(context.TODO(), iostream.New(ws))
		defer release()
	}()

	// Check the server reader for s1
	b1 := make([]byte, bufferSize)
	n, err := rs.Read(b1)
	require.NoError(t, err)
	assert.Equal(t, n, len(s1))
	assert.Equal(t, string(b1[0:n]), s1)

	// Write s2 to the client writer
	go func() {
		n, err := wc.Write([]byte(s2))
		require.NoError(t, err)
		assert.Equal(t, n, len(s2))
	}()

	// Check server reader for s2
	n, err = rs.Read(b1)
	require.NoError(t, err)
	assert.Equal(t, n, len(s2))
	assert.Equal(t, string(b1[0:n]), s2)

	// Closing the client reader should make the provider close the
	// server pipe writer (and therefore reader)
	rc.Close()
	_, err = rs.Read(make([]byte, 0))
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "EOF")
}

type mockCloser struct {
	Closed bool
	*bytes.Buffer
}

func (mc *mockCloser) Close() error {
	mc.Closed = true
	return nil
}
