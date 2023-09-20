//go:build wasm || tinygo.wasm || wasi
// +build wasm tinygo.wasm wasi

package system

//go:wasmimport ww __sock_close
//go:noescape
func sockClose() uint32
