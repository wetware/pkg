package system

import (
	"io"
)

const (
	_       = iota
	SockINT // interrupt
	SockEOF // end of file
)

type errno int32

func (e errno) Error() string {
	switch e {
	case SockINT:
		return "interrupt"

	case SockEOF:
		return io.EOF.Error()
	}

	panic(e)
}

func (e errno) Errno() int32 {
	return int32(e)
}
