//go:build wasm || tinygo.wasm || wasi
// +build wasm tinygo.wasm wasi

package system

import "capnproto.org/go/capnp/v3/exp/bufferpool"

//go:wasmimport ww __sock_close
//go:noescape
func sockClose() uint32

//go:wasmimport ww __sock_send
//go:noescape
func sockSend(offset, length uint32) uint32

//go:wasm-module ww
//go:export __sock_alloc
func sockAlloc(size uint32) uint32 {
	seg := alloc(size)
	return seg.offset
}

func alloc(size uint32) segment {
	buf := bufferpool.Default.Get(int(size))
	seg := segment{
		offset: bytesToPointer(buf),
		length: size,
	}
	exports[seg] = buf
	return seg
}

//go:wasm-module ww
//go:export __sock_notify
func sockNotify(offset, size uint32) {
	seg := segment{offset, size}
	incoming <- seg // TODO:  timeout
}
