//go:build wasm || tinygo.wasm || wasi
// +build wasm tinygo.wasm wasi

package system

//go:wasmimport ww __sock_close
//go:noescape
func sockClose() uint32

//go:wasmimport ww __sock_read
//go:noescape
func sockRead(offset, length uint32, timeout int64) uint32

//go:wasmimport ww __sock_write
//go:noescape
func sockWrite(offset, length uint32, timeout int64) uint32
