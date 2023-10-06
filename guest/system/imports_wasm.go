//go:build wasm || tinygo.wasm || wasi
// +build wasm tinygo.wasm wasi

package system

//go:wasmimport ww _sysclose
//go:noescape
func sysclose() uint32

//go:wasmimport ww _sysread
//go:noescape
func sysread(offset, length, size uint32) uint32

//go:wasmimport ww _syswrite
//go:noescape
func syswrite(offset, length, consumed uint32) uint32
