package unix_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/ocap/process/unix"
)

var server unix.Server

func TestStdin(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec := server.Executor()

	expected := "hello world"
	stdin := strings.NewReader(expected)
	stdout := new(bytes.Buffer)

	cmd := unix.Command(nil, "cat").
		Bind(unix.Stdin(stdin)).
		Bind(unix.Stdout(stdout))

	p := exec.Exec(ctx, cmd)
	defer p.Release()

	err := p.Wait(ctx)
	require.NoError(t, err)

	require.EqualValues(t, expected, stdout.String())
}

func TestStdout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec := server.Executor()

	expected := "hello world"
	stdout := new(bytes.Buffer)

	cmd := unix.Command(nil, "echo", "-n", expected).
		Bind(unix.Stdout(stdout))

	p := exec.Exec(ctx, cmd)
	defer p.Release()

	err := p.Wait(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, stdout.String())
}

func TestStderr(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec := server.Executor()

	expected := "hello world"
	stderr := new(bytes.Buffer)

	subcmd := fmt.Sprintf(`echo '%s' 1>&2`, expected)
	cmd := unix.Command(nil, "sh", "-c", subcmd).
		Bind(unix.Stderr(stderr))

	p := exec.Exec(ctx, cmd)
	defer p.Release()

	err := p.Wait(ctx)
	require.NoError(t, err)

	require.Equal(t, expected+"\n", stderr.String())
}
