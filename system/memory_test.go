package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSegment(t *testing.T) {
	t.Parallel()
	t.Helper()

	seg := segment{
		offset: 42,
		length: 9001,
	}

	t.Run("LoadStoreObject", func(t *testing.T) {
		t.Parallel()

		buf := make([]byte, seg.ObjectSize())
		seg.StoreObject(nil, buf)
		got := segment{}.LoadObject(nil, buf)

		require.Equal(t, seg, got,
			"loaded object should equal stored object")
	})

	t.Run("LoadStoreValue", func(t *testing.T) {
		t.Parallel()

		stack := make([]uint64, 1)
		seg.StoreValue(nil, stack)
		got := segment{}.LoadValue(nil, stack)

		require.Equal(t, seg, got,
			"loaded object should equal stored object")
	})
}
