package cluster

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/urfave/cli/v2"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/vat"
)

const killTimeout = 30 * time.Second

func run() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "run a WASM module on a cluster node",
		ArgsUsage: "<path> (defaults to stdin)",
		Action:    runAction(),
	}
}

func runAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		ctx := c.Context

		// Load the name of the entry function and the WASM file containing the module to run
		src, err := bytecode(c)
		if err != nil {
			return err
		}

		// Set up the wetware client and dial into the cluster
		h, err := vat.DialP2P()
		if err != nil {
			return err
		}
		defer h.Close()

		bootstrap, err := newBootstrap(c, h)
		if err != nil {
			return fmt.Errorf("discovery: %w", err)
		}
		defer bootstrap.Close()

		session, err = vat.Dialer{
			Host:    h,
			Account: auth.SignerFromHost(h),
		}.DialDiscover(c.Context, bootstrap, c.String("ns"))
		if err != nil {
			return err
		}

		// Obtain an executor and spawn a process

		bCtx, err := csp.NewBootContext().
			WithArgs(c.Args().Slice()...).
			WithCaps(capnp.Client(session.CapStore))
		if err != nil {
			return err
		}

		proc, release := session.Exec.Exec(ctx, src, 0, bCtx.Cap())
		defer release()

		waitChan := make(chan error, 1)
		go func() {
			waitChan <- proc.Wait(ctx)
		}()
		select {
		case err = <-waitChan:
			return err
		case <-ctx.Done():
			killChan := make(chan error, 1)
			go func() { killChan <- proc.Kill(context.Background()) }()
			select {
			case err = <-killChan:
				return err
			case <-time.After(killTimeout):
				return err
			}
		}
	}
}

func bytecode(c *cli.Context) ([]byte, error) {
	if c.Args().Len() > 0 {
		return os.ReadFile(c.Args().First()) // file path
	}

	return io.ReadAll(c.App.Reader) // stdin
}
