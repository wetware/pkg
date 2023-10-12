package pipe_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/pkg/util/pipe"
)

func TestReadWrite(t *testing.T) {
	t.Parallel()

	const greeting = "Hello, Pipe!"

	left, right := pipe.New()
	n, err := left.Write([]byte(greeting))
	assert.Equal(t, len(greeting), n, "should write full message")
	require.NoError(t, err, "should write message without error")

	var buf bytes.Buffer
	nn, err := io.Copy(&buf, right)
	assert.Equal(t, int64(n), nn, "should copy full message")
	require.ErrorIs(t, err, pipe.ErrInterrupt, "copy should return io interrupt")
	require.Equal(t, greeting, buf.String())

	const farewell = "Goodbye, Pipe!"

	n, err = right.Write([]byte(farewell))
	assert.Equal(t, len(farewell), n, "should write full message")
	require.NoError(t, err, "should write message without error")

	buf.Reset()
	nn, err = io.Copy(&buf, left)
	assert.Equal(t, int64(n), nn, "should copy full message")
	require.ErrorIs(t, err, pipe.ErrInterrupt, "copy should return io interrupt")
	require.Equal(t, farewell, buf.String())
}

func BenchmarkPipe(b *testing.B) {
	message := []byte("this is a small message intended for benchmarking the pipe")

	left, right := pipe.New()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		left.Write(message)
		right.Read(message)
	}
}
