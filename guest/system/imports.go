//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi
// +build !purego,!appengine,!wasm,!tinygo.wasm,!wasi

package system

// func vatID() uint64        { return 0 }
func sockClose() int32 {
	return 0
}

func sockSend(offset, length uint32) int32 {
	return 0
}
