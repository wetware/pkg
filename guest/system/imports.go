//go:build wasm || tinygo.wasm || wasi
// +build wasm tinygo.wasm wasi

package system

/*
	This file contains the host imports for a WASM process.
*/

//go:wasmimport ww __poll
func pollHost() int32
