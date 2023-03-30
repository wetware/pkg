package process_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"github.com/wetware/ww/pkg/process"
)

func TestExecutor(t *testing.T) {
	t.Parallel()

	r := wazero.NewRuntime(context.Background())
	wasi_snapshot_preview1.MustInstantiate(context.Background(), r)

	exec := process.Server{Runtime: r}.Executor()
	defer exec.Release()

	proc, release := exec.Spawn(context.Background(), testdata())
	defer release()

	err := proc.Start(context.Background())
	require.NoError(t, err, "should start process")

	err = proc.Wait(context.Background())
	require.Error(t, err, "should return an error from process")

	ee, ok := err.(*sys.ExitError)
	require.True(t, ok, "should return sys.ExitError")
	assert.Equal(t, uint32(99), ee.ExitCode())
}

func testdata() []byte {
	b, err := os.ReadFile("testdata/main.wasm")
	if err != nil {
		panic(err)
	}

	return b
}
