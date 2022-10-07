package wasm_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tetratelabs/wazero"
	"github.com/wetware/ww/pkg/process/wasm"
)

func TestWASM(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("./testdata/hello.wasm")
	require.NoError(t, err)
	require.NotNil(t, src)

	config := wazero.NewRuntimeConfigCompiler()
	rf := wasm.RuntimeFactory{Config: config}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runctx := wasm.NewRunContext(src).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	p, release := rf.Runtime(ctx).Exec(ctx, runctx)
	defer release()

	f, release := p.Run(context.Background())
	defer release()

	assert.NoError(t, f.Err())
}
