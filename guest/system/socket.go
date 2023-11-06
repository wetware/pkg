package system

import (
	"io"
	"unsafe"

	"github.com/stealthrocket/wazergo/types"
)

type Socket struct{ io.Reader }

func (Socket) Write(p []byte) (n int, err error) {
	eno := sockSend(
		bytesToPointer(p),
		uint32(len(p)))
	return len(p), maybeError(eno)
}

func (Socket) Close() error {
	eno := sockClose()
	return maybeError(eno)
}

func maybeError(e int32) error {
	if e == 0 {
		return nil
	}

	return types.Errno(e)
}

//go:inline
func bytesToPointer(b []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
}
