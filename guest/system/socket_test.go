package system_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"runtime"
	"testing"

	"github.com/stealthrocket/wazergo"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/wetware/pkg/system"
)

func TestSocket(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	b, err := os.ReadFile("internal/main.wasm")
	require.NoError(t, err)
	require.NotEmpty(t, b)

	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true))
	defer r.Close(ctx)

	wasi.MustInstantiate(ctx, r)

	// Instantiate the system host module.
	sock := &mockSocket{}
	sys, err := system.Instantiate(ctx, r, sock)
	require.NoError(t, err)
	defer sys.Close(ctx)
	ctx = wazergo.WithModuleInstance(ctx, sys)

	cm, err := r.CompileModule(ctx, b)
	require.NoError(t, err)
	defer cm.Close(ctx)

	stdin := bytes.NewBufferString("Hello, Wetware!")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	mod, err := r.InstantiateModule(ctx, cm, wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithArgs("test-rom").
		WithName("test-rom").
		WithStdin(stdin).
		WithStdout(stdout).
		WithStderr(stderr))
	require.NoError(t, err)
	defer mod.Close(ctx)

	require.NoError(t, err)
	require.Equal(t, "Hello, Go!", sock.String())
	require.Empty(t, stderr.String(), "stderr should be empty")
}

type mockSocket struct{ bytes.Buffer }

func (s *mockSocket) Close() error { return nil }
