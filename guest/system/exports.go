package system

import (
	"unsafe"

	"capnproto.org/go/capnp/v3/exp/bufferpool"
)

var exports = map[segment][]byte{}

type segment struct {
	offset uintptr
	length int32
}

func export(b []byte) segment {
	seg := segmentFromBytes(b)
	exports[seg] = b
	return seg
}

func malloc(size int32) segment {
	if size > 0 {
		buf := bufferpool.Default.Get(int(size))
		return export(buf)
	}

	panic(size)
}

func free(seg segment) {
	buf := exports[seg]
	delete(exports, seg)
	bufferpool.Default.Put(buf)
}

func segmentFromBytes(b []byte) segment {
	return segment{
		offset: (uintptr)(unsafe.Pointer(unsafe.SliceData(b))),
		length: int32(len(b)),
	}
}

func segmentToBytes(seg segment) []byte {
	return exports[seg]
}
