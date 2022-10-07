package unix_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/pkg/process/unix"
)

var (
	server unix.Server
	ctx    = context.Background()
)

func TestStdin(t *testing.T) {
	t.Parallel()

	exec := server.Executor()
	defer exec.Release()

	expected := "hello world"
	stdin := strings.NewReader(expected)
	stdout := new(bytes.Buffer)

	cmd := unix.Command("cat").
		Bind(unix.Stdin(stdin)).
		Bind(unix.Stdout(stdout))

	p, release := exec.Exec(ctx, cmd)
	defer release()

	err := p.Wait(ctx)
	require.NoError(t, err)

	require.EqualValues(t, expected, stdout.String())
}

func TestStdout(t *testing.T) {
	t.Parallel()

	exec := server.Executor()
	defer exec.Release()

	expected := "hello world"
	stdout := new(bytes.Buffer)

	cmd := unix.Command("echo", "-n", expected).
		Bind(unix.Stdout(stdout))

	p, release := exec.Exec(ctx, cmd)
	defer release()

	err := p.Wait(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, stdout.String())
}

func TestStderr(t *testing.T) {
	t.Parallel()

	exec := server.Executor()
	defer exec.Release()

	expected := "hello world"
	stderr := new(bytes.Buffer)

	subcmd := fmt.Sprintf(`echo '%s' 1>&2`, expected)
	cmd := unix.Command("sh", "-c", subcmd).
		Bind(unix.Stderr(stderr))

	p, release := exec.Exec(ctx, cmd)
	defer release()

	err := p.Wait(ctx)
	require.NoError(t, err)

	require.Equal(t, expected+"\n", stderr.String())
}

func TestSignal(t *testing.T) {
	t.Parallel()
	t.Helper()

	/*
		Test supported signals.  These should all terminate the process.
	*/
	for _, sig := range []syscall.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
	} {
		// Use function closure to avoid shadowing 'sig'.
		func(sig syscall.Signal) {
			t.Run(sig.String(), func(t *testing.T) {
				t.Parallel()

				exec := server.Executor()
				defer exec.Release()

				p, release := exec.Exec(ctx, unix.Command("sleep", "5"))
				defer release()

				cherr := make(chan error, 1)
				go func() {
					cherr <- p.Wait(ctx)
				}()

				f, release := p.Signal(ctx, sig)
				defer release()

				err := f.Await(ctx)
				require.NoError(t, err)

				assert.EqualError(t, <-cherr,
					fmt.Sprintf("proc.capnp:Waiter.wait: signal: %s", sig),
					"should report '%s' signal", sig)
			})
		}(sig)
	}

	/*
		Test that we handle unrecognized signals correctly.
	*/
	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()

		// ensure the process is killed upon exiting the test func.
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		exec := server.Executor()
		defer exec.Release()

		p, release := exec.Exec(ctx, unix.Command("sleep", "5"))
		defer release()

		sig := syscall.SIGABRT // NOT supported

		f, release := p.Signal(ctx, sig)
		defer release()

		err := f.Await(ctx)
		require.EqualError(t, err,
			fmt.Sprintf("proc.capnp:Unix.Proc.signal: unknown signal: %#x", int(sig)),
			"should reject invalid signal")
	})
}
