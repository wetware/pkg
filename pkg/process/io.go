package process

import (
	"bytes"
	"io"

	iostream_api "github.com/wetware/ww/internal/api/iostream"
	"github.com/wetware/ww/pkg/iostream"
)

// processIo constains all the required components for a process to
// run.
type processIo struct {
	inR       *io.PipeReader
	inW       *io.PipeWriter
	outR      *io.PipeReader
	outW      *io.PipeWriter
	errBuffer *bytes.Buffer

	in  iostream_api.Stream
	out iostream_api.Provider
}

// newIo is the default constructor fo Io.
func newIo() processIo {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	errBuffer := new(bytes.Buffer)

	return processIo{
		inR:       inR,
		inW:       inW,
		outR:      outR,
		outW:      outW,
		errBuffer: errBuffer,

		in:  iostream_api.Stream(iostream.New(inW)),
		out: iostream_api.Provider(iostream.NewProvider(outR)),
	}
}
