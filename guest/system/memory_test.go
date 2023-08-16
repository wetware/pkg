package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExports(t *testing.T) {
	t.Parallel()

	const length = 10

	offset := __alloc(length)
	seg := segment{offset: pointer(offset), size: size(length)}
	require.Contains(t, exports, seg, "should contain freed segment")

	__free(uint32(seg.offset), uint32(seg.size))
	require.NotContains(t, exports, seg, "should not contain freed segment")
}
