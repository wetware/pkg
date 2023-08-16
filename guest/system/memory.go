package system

import (
	"unsafe"

	"capnproto.org/go/capnp/v3/exp/bufferpool"
)

var exports = map[segment][]byte{}

type (
	pointer uint32
	size    uint32
)

type segment struct {
	offset pointer
	size   size
}

func alloc(s size) segment {
	buf := bufferpool.Default.Get(int(s))
	seg := pointerTo(buf)
	exports[seg] = buf
	return seg
}

//go:export __ww_guest_alloc
func __alloc(length uint32) uint32 {
	seg := alloc(size(length))
	return uint32(seg.offset)
}

func free(seg segment) {
	buf := exports[seg]
	delete(exports, seg)
	bufferpool.Default.Put(buf)
}

//go:export __ww_guest_free
func __free(offset, length uint32) {
	free(segment{
		offset: pointer(offset),
		size:   size(length),
	})
}

func pointerTo(buf []byte) segment {
	return segment{
		offset: bytesToPointer(buf),
		size:   size(len(buf)),
	}
}

//go:inline
func bytesToPointer(s []byte) pointer {
	offset := unsafe.SliceData(s)
	return (*(*pointer)(unsafe.Pointer(offset)))
}
