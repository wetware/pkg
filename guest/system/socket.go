package system

import (
	"io"
	"log/slog"
	"time"
	"unsafe"

	"github.com/stealthrocket/wazergo/types"
	"github.com/wetware/pkg/system"
)

type socket struct{}

func Socket() io.ReadWriteCloser {
	return socket{}
}

func (socket) Read(b []byte) (n int, err error) {
	var u uint32
	for {
		eno := sysread(
			bytesToPointer(b), // buffer offset
			uint32(len(b)),    // buffer size
			uint32ToPointer(&u))
		n += int(u)
		b = b[n:]

		slog.Debug("read data to system socket",
			"bytes", u,
			"status", eno)

		switch eno {
		case system.SockEOF:
			err = io.EOF // all data was read; we're done!

		case system.SockINT: // wait for the writer to have written data
			time.Sleep(time.Millisecond)
			continue

		default:
			err = types.Errno(eno)
		}

		return
	}
}

// Write is a blocking operation.
func (socket) Write(b []byte) (n int, err error) {
	var u uint32
	for {
		eno := syswrite(
			bytesToPointer(b), // buffer offset
			uint32(len(b)),    // buffer size
			uint32ToPointer(&u))
		n += int(u)
		b = b[n:]

		slog.Debug("wrote data to system socket",
			"bytes", u,
			"status", eno)

		switch eno {
		case system.SockEOF:
			err = io.EOF

		case system.SockINT:
			time.Sleep(time.Millisecond)
			continue

		default:
			err = types.Errno(eno)
		}

		return
	}
}

func (socket) Close() error {
	slog.Debug("guest socket closed")

	if eno := sysclose(); eno != 0 {
		return types.Errno(eno)
	}

	return nil
}

//go:inline
func bytesToPointer(b []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
}

//go:inline
func uint32ToPointer(u *uint32) uint32 {
	return uint32(uintptr(unsafe.Pointer(u)))
}
