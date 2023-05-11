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

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/ww"
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
		Action:               action(),
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
			fx.Supply(c, c.String("ns")),
			system.WithDefaultServer(),
			fx.Populate(&wetware /*&code*/),
			fx.Decorate(bytecode),
			fx.Provide(
				stdio,
				logger,
				identity))
		if err := start(app); err != nil {
			return err
		}

		if err := wetware.Exec(c.Context); err != nil {
			return err
		}

		return stop(app)
	}
}

func logger(c *cli.Context) log.Logger {
	return log.New()
}

type Stdio struct {
	fx.Out

	Stdin  io.Reader `name:"stdin"`
	Stdout io.Writer `name:"stdout"`
	Stderr io.Writer `name:"stderr"`
}

func stdio(c *cli.Context) Stdio {
	return Stdio{
		Stdin:  c.App.Reader,
		Stdout: c.App.Writer,
		Stderr: c.App.ErrWriter,
	}
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
