/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/wetware/ww"
	cmdrun "github.com/wetware/ww/internal/cmd/run"
	cmdstart "github.com/wetware/ww/internal/cmd/start"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGKILL)
	defer cancel()

	app := &cli.App{
		Name:                 "wetware",
		Version:              ww.Version,
		HelpName:             "ww",
		Usage:                "simple, secure clusters",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cmdrun.Command(),
			cmdstart.Command(),
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1) // application error
	}
}
