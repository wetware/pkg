package repo

import "github.com/urfave/cli/v2"

// Commands under `repo`
func Commands() []*cli.Command {
	return []*cli.Command{{
		Name:   "init",
		Usage:  "initialize a repository",
		Flags:  initFlags(),
		Action: initAction(),
	}}
}
