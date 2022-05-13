package proc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/cap/proc"
)

func TestStdout(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := proc.Server{}
	client := server.NewClient()

	expected := "hello world"
	cmd, release := client.Command(ctx, "echo", expected)
	defer release()

	stdout, release := cmd.StdoutPipe(ctx)
	defer release()

	err := cmd.Start(ctx)
	require.NoError(t, err)

	result := make([]byte, 4096)
	n, err := stdout.Read(ctx, result)

	require.NoError(t, err)
	require.EqualValues(t, expected, string(result[:n-1])) // -1 is to remove the newline
}

func TestStderr(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := proc.Server{}
	client := server.NewClient()

	expected := "hello world"
	cmd, release := client.Command(ctx, "sh", "-c", fmt.Sprintf("echo %s 1>&2", expected))
	defer release()

	stderr, release := cmd.StderrPipe(ctx)
	defer release()

	err := cmd.Start(ctx)
	require.NoError(t, err)

	result := make([]byte, 4096)
	n, err := stderr.Read(ctx, result)

	require.NoError(t, err)
	require.EqualValues(t, expected, string(result[:n-1])) // -1 is to remove the newline
}

func TestStdin(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := proc.Server{}
	client := server.NewClient()

	expected := []byte("hello world")
	cmd, release := client.Command(ctx, "cat")
	defer release()

	stdin, release := cmd.StdinPipe(ctx)
	defer release()

	stdout, release := cmd.StdoutPipe(ctx)
	defer release()

	err := cmd.Start(ctx)
	require.NoError(t, err)

	n, err := stdin.Write(ctx, expected)
	require.NoError(t, err)
	require.Equal(t, len(expected), n)

	result := make([]byte, 4096)
	n, err = stdout.Read(ctx, result)

	require.NoError(t, err)
	require.EqualValues(t, expected, result[:n]) // -1 is to remove the newline
}
