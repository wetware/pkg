package wasm_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/wetware/ww/pkg/process/wasm"
)

func TestWASM(t *testing.T) {
	t.Parallel()

	b, err := os.ReadFile("./testdata/hello.wasm")
	require.NoError(t, err)
	require.NotNil(t, b)

	config := wazero.NewRuntimeConfigCompiler()
	f := wasm.RuntimeFactory{Config: config}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fut, release := f.Runtime(ctx).Exec(ctx, b)
	defer release()

	require.NoError(t, fut.Err())
}
