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

	buf := new(bytes.Buffer)
	stream := iostream.New(buf, nil)

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
}
