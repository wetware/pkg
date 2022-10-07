package process_test

import (
	"context"
	"testing"

	"capnproto.org/go/capnp/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/internal/api/proc"
	"github.com/wetware/ww/pkg/process"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	_, seg := capnp.NewSingleSegmentMessage(nil)
	ps, err := proc.NewRootExecutor_exec_Params(seg)
	require.NoError(t, err)

	config := process.NewConfig(mockType()).
		Bind(mockParam)

	err = config(ps)
	require.NoError(t, err, "configuration should succeed")

	ptr, err := ps.Config()
	require.NoError(t, err)

	// HACK:  we used a results struct as a mock config type
	conf := proc.Executor_exec_Results(ptr.Struct())
	ok := conf.Proc().IsValid()
	assert.True(t, ok, "should have set parameter")
}

// HACK:  use results struct as a mock config
func mockType() process.ConfigType[proc.Executor_exec_Results] {
	return func(a capnp.Arena) (proc.Executor_exec_Results, error) {
		_, seg := capnp.NewSingleSegmentMessage(nil)
		return proc.NewRootExecutor_exec_Results(seg)
	}
}

func mockParam(ps proc.Executor_exec_Results) error {
	return ps.SetProc(proc.Waiter_ServerToClient(mockProc{}))
}

type mockProc struct{}

func (mockProc) Wait(context.Context, proc.Waiter_wait) error {
	return nil
}
