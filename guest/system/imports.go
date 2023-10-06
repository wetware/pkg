//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi
// +build !purego,!appengine,!wasm,!tinygo.wasm,!wasi

package system

func sysclose() int32 {
	return 0
}

func sysread(offset, length, consumed uint32) uint32 {
	return 0
}

func syswrite(offset, length, consumed uint32) uint32 {
	return 0
}
