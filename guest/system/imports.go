//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi
// +build !purego,!appengine,!wasm,!tinygo.wasm,!wasi

package system

func sockClose() int32 {
	return 0
}

func sockSend(offset, length uint32) uint32 {
	return 0
}
