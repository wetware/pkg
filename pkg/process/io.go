package process

import (
	"context"
	"errors"
	"io"

	api "github.com/wetware/ww/internal/api/iostream"
	"github.com/wetware/ww/pkg/iostream"
)

// processIO constains all the required components for a process to
// run.
type processIO struct {
	inR  *io.PipeReader
	inW  *io.PipeWriter
	outR *io.PipeReader
	outW *io.PipeWriter
	errR *io.PipeReader
	errW *io.PipeWriter
}

// newIO is the default constructor fo Io.
func newIO() processIO {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()

	return processIO{
		inR:  inR,
		inW:  inW,
		outR: outR,
		outW: outW,
		errR: errR,
		errW: errW,

		// in:  iostream_api.Stream(iostream.New(inW)),
		// out: iostream_api.Provider(iostream.NewProvider(outR)),
		// err: iostream_api.Provider(iostream.NewProvider(errR)),
	}
}

func (pio processIO) Stdin() api.Stream {
	return api.Stream(iostream.New(pio.inW))
}

func (pio processIO) BindStdout(ctx context.Context, s api.Stream) error {
	// return iostream.Stream(s)
	return errors.New("NOT IMPLEMENTED")
}

func (pio processIO) BindStderr(ctx context.Context, s api.Stream) error {
	// return iostream.Stream(s)
	return errors.New("NOT IMPLEMENTED")
}

func (pio processIO) Release() {
	pio.errW.Close()
	pio.outW.Close()
}
