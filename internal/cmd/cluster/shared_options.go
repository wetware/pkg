package cluster

import (
	"github.com/urfave/cli/v2"
)

const _json = "json"

var (
	boolFlag = cli.BoolFlag{
		Name:    _json,
		Usage:   "print results as json",
		EnvVars: []string{"WW_FMT_JSON"},
	}
)
