package keygen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var descr = `Generates a 256-bit, base16-encoded, cryptographically symmetric key.

PROTOCOL:
	/key/swarm/psk/1.0.0/

ENCODING:
	/base16/`

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:      "output",
		Aliases:   []string{"out", "o"},
		Usage:     "write key to file",
		TakesFile: true,
	},
}

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:        "keygen",
		Usage:       "generate a shared secret for a cluster",
		Description: descr,
		Flags:       flags,
		Action:      run(),
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return errors.Wrap(err, "read random")
		}

		w, err := getWriter(c)
		if err != nil {
			return err
		}
		defer w.Close()

		// writing to a file can fail unexpectedly, so handle the error.
		_, err = fmt.Fprint(w, hex.EncodeToString(key))
		return errors.Wrap(err, "fwrite")
	}
}

func getWriter(c *cli.Context) (io.WriteCloser, error) {
	if c.String("out") != "" {
		path := filepath.Clean(c.String("out"))

		// Open a write-only file, failing if one already exists.  Set the SYNC flag
		// to reduce the chance of flush-errors when calling Close (this avoids having
		// to check the error in a `defer` statement).
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0740)
		if err != nil {
			return nil, errors.Wrap(err, "fopen")
		}

		return f, nil
	}

	return nopCloser{c.App.Writer}, nil
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }
