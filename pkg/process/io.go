package process

import (
	"context"
	"io"

	iostream_api "github.com/wetware/ww/internal/api/iostream"
	"github.com/wetware/ww/pkg/iostream"
)

// processIo constains all the required components for a process to
// run.
type processIo struct {
	inR  *io.PipeReader
	inW  *io.PipeWriter
	outR *io.PipeReader
	outW *io.PipeWriter
	errR *io.PipeReader
	errW *io.PipeWriter

	in  iostream_api.Stream
	out iostream_api.Provider
	err iostream_api.Provider
}

// newIo is the default constructor fo Io.
func newIo() processIo {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()

	return processIo{
		inR:  inR,
		inW:  inW,
		outR: outR,
		outW: outW,
		errR: errR,
		errW: errW,

		in:  iostream_api.Stream(iostream.New(inW)),
		out: iostream_api.Provider(iostream.NewProvider(outR)),
		err: iostream_api.Provider(iostream.NewProvider(errR)),
	}
}

func (pio processIo) closeWriters(ctx context.Context) {
	defer pio.outW.Close()
	defer pio.errW.Close()
	defer pio.in.Close(ctx, nil)
}
