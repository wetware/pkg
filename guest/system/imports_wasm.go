//go:build wasm || tinygo.wasm || wasi
// +build wasm tinygo.wasm wasi

package system

// //go:wasmimport ww vat_id
// //go:noescape
// func vatID() uint64

//go:wasmimport ww sock_close
//go:noescape
func sockClose() int32

//go:wasmimport ww sock_send
//go:noescape
func sockSend(offset, length uint32) int32
