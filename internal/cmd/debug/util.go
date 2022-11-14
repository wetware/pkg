package debug

import (
	"errors"
	"os"

	"github.com/urfave/cli/v2"
)

func writer(c *cli.Context) (*os.File, error) {
	if c.Bool("stdout") {
		return os.Stdout, nil
	}

	if c.IsSet("out") {
		return os.Create(c.Path("out"))
	}

	return nil, errors.New("must pass -out or -stdout")
}
