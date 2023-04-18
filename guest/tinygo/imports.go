//go:build wasm || tinygo.wasm || wasi

package ww

//go:wasm-module ww
//export __host_read
func hostRead(offset, length uint32, n *uint32) int32

//go:wasm-module ww
//export __host_write
func hostWrite(offset, length uint32, n *uint32) int32

//go:wasm-module ww
//go:export __host_close
func hostClose() int32
