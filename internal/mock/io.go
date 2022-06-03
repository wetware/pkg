//go:generate mockgen -source=io.go -destination=io/io.go -package=mock_io
package mock

import "io"

type (
	Reader interface{ io.Reader }
	Writer interface{ io.Writer }
	Closer interface{ io.Closer }

	ReadWriter      interface{ io.ReadWriter }
	ReadCloser      interface{ io.ReadCloser }
	WriteCloser     interface{ io.WriteCloser }
	ReadWriteCloser interface{ io.ReadWriteCloser }

	WriterTo   interface{ io.WriterTo }
	ReaderFrom interface{ io.ReaderFrom }
)
