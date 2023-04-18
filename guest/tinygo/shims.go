//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi

package ww

/*

	shims.go contains shim functions for WASM imports, which allows
	symbol names to resolve on non-WASM architectures.

*/

func hostRead(offset, length uint32, n *uint32) int32 {
	return 0
}

func hostWrite(offset, length uint32, n *uint32) int32 {
	return 0
}

func hostClose() int32 {
	return 0
}
