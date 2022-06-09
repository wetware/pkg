package unix_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/ocap/process/unix"
)

var server unix.Server

func TestStdout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec := server.Executor()

	expected := "hello world"
	buf := new(bytes.Buffer)

	cmd := unix.Command(nil, "echo", "-n", expected).
		Bind(unix.Stdout(buf))

	p, release := exec.Exec(ctx, cmd)
	defer release()

	err := p.Wait(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, buf.String())
}

func TestStderr(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec := server.Executor()

	expected := "hello world"
	buf := new(bytes.Buffer)

	subcmd := fmt.Sprintf(`echo '%s' 1>&2`, expected)
	cmd := unix.Command(nil, "sh", "-c", subcmd).
		Bind(unix.Stderr(buf))

	p, release := exec.Exec(ctx, cmd)
	defer release()

	err := p.Wait(ctx)
	require.NoError(t, err)

	require.Equal(t, expected+"\n", buf.String())
}

// func TestStdin(t *testing.T) {
// 	t.Parallel()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	exec := server.Executor()

// 	expected := []byte("hello world")
// 	buf := new(bytes.Buffer)

// 	cmd := unix.Command(nil, "cat").
// 		Bind(unix.Stdout(buf))

// 	p, release := exec.Exec(ctx, cmd)
// 	defer release()

// 	stdin, release := p.Stdin(ctx)
// 	defer release()

// 	f, release := stdin.Write(ctx, expected)
// 	defer release()

// 	require.NoError(t, f.Err())
// 	require.EqualValues(t, expected, buf.Bytes())
// }
