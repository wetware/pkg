package debug

import (
	_ "embed"
	"encoding/json"
	"html/template"

	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/debug"
)

//go:embed sysinfo.tmpl
var sysinfo string

func info() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "return debug info about a host",
		ArgsUsage: "<peer>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Usage:   "print results as json",
				EnvVars: []string{"WW_FMT_JSON"},
			},
		},
		Action: querySysInfo(),
	}
}

func querySysInfo() cli.ActionFunc {
	return func(c *cli.Context) error {
		// a, release := node.Walk(c.Context, target(c))
		// defer release()

		// d, release := anchor.Host(a).Debug(c.Context)
		// defer release()

		// TEST
		d, release := node.Debug(c.Context)
		defer release()
		// -- TEST

		var info debug.SysInfo
		if err := d.SysInfo(c.Context, &info); err != nil {
			return err
		}

		return renderSysInfo(c, info)
	}
}

func renderSysInfo(c *cli.Context, info debug.SysInfo) error {
	if c.Bool("json") {
		return json.NewEncoder(c.App.Writer).Encode(info)
	}

	t := template.Must(template.New("sysinfo").Parse(sysinfo))
	return t.Execute(c.App.Writer, info)
}
