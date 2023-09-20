//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi
// +build !purego,!appengine,!wasm,!tinygo.wasm,!wasi

package system

func sockClose() int32 {
	return 0
}

func sockRead(offset, length uint32, timeout int64) uint32 {
	return 0
}

func sockWrite(offset, length uint32, timeout int64) uint32 {
	return 0
}
