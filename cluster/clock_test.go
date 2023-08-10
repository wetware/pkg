package cluster_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/cluster"
)

func TestClock(t *testing.T) {
	t.Parallel()

	c := cluster.NewClock(time.Millisecond)
	defer c.Stop()

	require.NotNil(t, c, "should return clock")
	require.NotNil(t, c.Context(), "should return non-nil context")

	select {
	case <-c.Context().Done():
		t.Fatal("should not return expired context")
	default:
	}

	select {
	case <-c.Tick():
	case <-time.After(time.Millisecond * 10):
		t.Fatal("should produce ticks")
	}

	c.Stop()

	select {
	case <-c.Context().Done():
	case <-c.Tick():
		t.Fatal("should not close tick chan before context expires")
	}

	select {
	case <-time.After(time.Millisecond * 10):
	case <-c.Tick():
		t.Fatal("should stop producing ticks")
	}
}
