/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/ww"
	cmdstart "github.com/wetware/ww/internal/cmd/start"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/pkg/server"
	"github.com/wetware/ww/system"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.StringFlag{
		Name:    "rom",
		Usage:   "cid of boot rom",
		EnvVars: []string{"WW_ROM"},
	},
	&cli.BoolFlag{
		Name:    "stdin",
		Aliases: []string{"s"},
		Usage:   "load system image from stdin",
	},
	&cli.BoolFlag{
		Name:  "debug",
		Usage: "enable debug logging",
	},
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGKILL)
	defer cancel()

	app := &cli.App{
		Name:                 "wetware",
		HelpName:             "ww",
		Usage:                "simple, secure clusters",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		EnableBashCompletion: true,
		Flags:                flags,
		Commands: []*cli.Command{
			cmdstart.Command(),
		},
		Action: action(),
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1) // application error
	}
}

func action() cli.ActionFunc {
	return func(c *cli.Context) error {
		var wetware ww.Ww

		app := fx.New(fx.NopLogger,
			fx.Supply(c, new(anchor.Node)),
			fx.Provide(env, identity),
			fx.Decorate(bytecode),
			fx.Populate(&wetware),
			system.WithDefaultServer(),
			fx.Invoke(joinCluster))
		if err := start(app); err != nil {
			return err
		}

		if err := wetware.Exec(c.Context); err != nil {
			return err
		}

		return stop(app)
	}
}

type Env struct {
	fx.Out

	Context context.Context
	Log     log.Logger

	NS string

	Stdin  io.Reader `name:"stdin"`
	Stdout io.Writer `name:"stdout"`
	Stderr io.Writer `name:"stderr"`
}

func env(c *cli.Context) Env {
	return Env{
		Context: c.Context,
		Log:     logger(c),
		NS:      c.String("ns"),
		Stdin:   c.App.Reader,
		Stdout:  c.App.Writer,
		Stderr:  c.App.ErrWriter,
	}
}

func logger(c *cli.Context) log.Logger {
	level := log.InfoLevel
	if c.Bool("debug") {
		level = log.DebugLevel
	}

	return log.New(log.WithLevel(level))
}

func identity(c *cli.Context) (crypto.PrivKey, error) {
	privkey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	return privkey, err
}

func bytecode(c *cli.Context, rom system.ROM) (system.ROM, error) {
	// user specified the CID of a ROM?
	if c.IsSet("rom") {
		panic("TODO:  load ROM from BitSwap") // FIXME
	}

	if c.Bool("stdin") {
		return io.ReadAll(c.App.Reader)
	}

	// file?
	if c.Args().Len() > 0 {
		return os.ReadFile(c.Args().First())
	}

	// use the default bytecode, provided by Fx.
	return rom, nil
}

func joinCluster(lx fx.Lifecycle, vat casm.Vat, j server.Joiner, ps *pubsub.PubSub) {
	var n *server.Node
	lx.Append(fx.StartStopHook(
		func(ctx context.Context) (err error) {
			if n, err = j.Join(vat, ps); err == nil {
				err = n.Bootstrap(ctx)
			}

			return
		},
		func() error {
			return n.Close()
		},
	))
}

func start(app *fx.App) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		app.StartTimeout())
	defer cancel()

	if err := app.Start(ctx); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	return nil
}

func stop(app *fx.App) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		app.StartTimeout())
	defer cancel()

	if err := app.Stop(ctx); err != nil {
		return fmt.Errorf("stop: %w", err)
	}

	return nil
}
