package wasm_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tetratelabs/wazero"
	"github.com/wetware/ww/pkg/csp/wasm"
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

	var buf bytes.Buffer
	runctx := wasm.NewContext(src).
		WithStdout(&buf)

	p, release := rf.Runtime(ctx).Exec(ctx, runctx)
	defer release()

	f, release := p.Run(context.Background())
	defer release()

	require.NoError(t, f.Err())
	require.Equal(t, "hello, WASM!\n", buf.String())
}
