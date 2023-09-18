package cluster

import (
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v2"
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
		return nil

		// // Load the name of the entry function and the WASM file containing
		// // the module to run.
		// rom, err := bytecode(c)
		// if err != nil {
		// 	return err
		// }

		// // Set up the wetware client and dial into the cluster.
		// h, err := vat.DialP2P()
		// if err != nil {
		// 	return err
		// }
		// defer h.Close()

		// // Connect to peers.
		// bootstrap, err := newBootstrap(c, h)
		// if err != nil {
		// 	return fmt.Errorf("discovery: %w", err)
		// }
		// defer bootstrap.Close()

		// // Login into the wetware cluster.
		// sess, err = vat.Dialer{
		// 	Host:    h,
		// 	Account: auth.SignerFromHost(h),
		// }.DialDiscover(c.Context, bootstrap, c.String("ns"))
		// if err != nil {
		// 	return err
		// }

		// view := sess.Cluster()
		// defer view.Release()

		// // Prepare argv for the process.
		// args := []string{}
		// if c.Args().Len() > 1 {
		// 	args = append(args, c.Args().Slice()[1:]...)
		// }

		// // Run remote process.
		// // TODO: find a way to bind an anchor to an execution context.
		// proc, release := host.Exec().Exec(c.Context, core.Session(sess), rom, 0, args...)
		// defer release()

		// // Wait for remote process to end.
		// waitChan := make(chan error, 1)
		// go func() {
		// 	waitChan <- proc.Wait(c.Context)
		// }()
		// select {
		// case err = <-waitChan:
		// 	return err
		// case <-c.Context.Done():
		// 	killChan := make(chan error, 1)
		// 	go func() { killChan <- proc.Kill(context.Background()) }()
		// 	select {
		// 	case err = <-killChan:
		// 		return err
		// 	case <-time.After(killTimeout):
		// 		return err
		// 	}
		// }
	}
}

func bytecode(c *cli.Context) ([]byte, error) {
	if c.Args().Len() > 0 {
		return os.ReadFile(c.Args().First()) // file path
	}

	return io.ReadAll(c.App.Reader) // stdin
}
