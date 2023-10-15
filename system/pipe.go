package system

import (
	"log/slog"
	"time"

	"github.com/lthibault/iopipes"
)

func Pipe() (Socket, Socket) {
	lr, lw := iopipes.DrainingPipe(1<<13, nil)
	rr, rw := iopipes.DrainingPipe(1<<13, nil)

	left := pipe{
		pipeReader: pipeReader{lr},
		pipeWriter: pipeWriter{rw},
	}

	right := pipe{
		pipeReader: pipeReader{rr},
		pipeWriter: pipeWriter{lw},
	}

	return left, right
}

type pipe struct {
	pipeReader
	pipeWriter
}

func (p pipe) Close() error {
	return p.pipeWriter.Close()
}

type pipeReader struct {
	*iopipes.DrainingPipeReader
}

func (p pipeReader) SetReadDeadline(t time.Time) error {
	slog.Warn("SetReadDeadline:  NOT IMPLEMENTED")
	return nil
}

type pipeWriter struct {
	*iopipes.DrainingPipeWriter
}

func (p pipeWriter) SetWriteDeadline(t time.Time) error {
	slog.Warn("SetWriteDeadline:  NOT IMPLEMENTED")
	return nil
}
