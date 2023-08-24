package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSegment(t *testing.T) {
	t.Parallel()

	seg := segment{
		offset: 42,
		length: 9001,
	}

	buf := make([]byte, seg.ObjectSize())
	seg.StoreObject(nil, buf)

	got := segment{}.LoadObject(nil, buf)
	require.Equal(t, seg, got,
		"loaded object should equal stored object")
}
