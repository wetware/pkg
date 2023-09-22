package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWrite(t *testing.T) {
	t.Parallel()

	p := newPipe(16)

	t.Run("Write", func(t *testing.T) {
		n, err := p.Write([]byte("hello, world!"))
		assert.Equal(t, len("hello, world!"), n, "should write entire payload")
		require.NoError(t, err, "write should succeed")

		n, err = p.Write([]byte("this will fail"))
		assert.Equal(t, 3, n, "should overflow after exactly 3 bytes have been written")
		require.Error(t, err, "error should not be nil")
		require.ErrorIs(t, err, ErrOverflow, "error should indicate buffer overflow")
	})

	t.Run("Read", func(t *testing.T) {
		buf := make([]byte, len("hello, world!"))
		n, err := p.Read(buf)
		require.NoError(t, err, "read should succeed")
		assert.Equal(t, "hello, world!", string(buf[:n]))
	})

	assert.NoError(t, p.Close(), "should close without error")
}
