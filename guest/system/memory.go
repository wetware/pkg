package system

import (
	"unsafe"

	"capnproto.org/go/capnp/v3/exp/bufferpool"
	"capnproto.org/go/capnp/v3/exp/spsc"
)

var (
	input   spsc.Queue[[]byte]
	exports = map[segment][]byte{}
)

//export handler
func onMessage(offset, length uint32) {
	seg := segment{
		offset: offset,
		length: length,
	}

	input.Send(exports[seg])
	delete(exports, seg)
}

type segment struct {
	offset, length uint32
}

func export(b []byte) segment {
	seg := segmentFromBytes(b)
	exports[seg] = b
	return seg
}

//export alloc
func alloc(size uint32) segment {
	if size > 0 {
		buf := bufferpool.Default.Get(int(size))
		return export(buf)
	}

	panic(size)
}

func segmentFromBytes(b []byte) segment {
	return segment{
		offset: uint32((uintptr)(unsafe.Pointer(unsafe.SliceData(b)))),
		length: uint32(len(b)),
	}
}

// func segmentToBytes(seg segment) []byte {
// 	return exports[seg]
// }
