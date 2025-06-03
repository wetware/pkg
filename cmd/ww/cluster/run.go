package cluster

import (
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	api "github.com/wetware/pkg/api/core"
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
		// Load the name of the entry function and the WASM file containing
		// the module to run.
		rom, err := bytecode(c)
		if err != nil {
			return err
		}

		// Prepare argv for the process.
		args := []string{}
		if c.Args().Len() > 1 {
			args = append(args, c.Args().Slice()[1:]...)
		}

		// Get a session.
		h, err := vat.DialP2P()
		if err != nil {
			return err
		}
		sess, close, err := BootstrapSession(c, h)
		defer close()
		if err != nil {
			return err
		}

		p, release := sess.Exec().Exec(c.Context, api.Session(sess), rom, 0, args...)
		defer release()
		return p.Wait(c.Context)
	}
}

func bytecode(c *cli.Context) ([]byte, error) {
	if c.Args().Len() > 0 {
		return os.ReadFile(c.Args().First()) // file path
	}

	return io.ReadAll(c.App.Reader) // stdin
}
