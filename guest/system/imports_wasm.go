//go:build wasm || tinygo.wasm || wasi
// +build wasm tinygo.wasm wasi

package system

import "github.com/stealthrocket/wazergo/types"

//go:wasmimport ww __send
func send(offset, length uint32) types.Errno
