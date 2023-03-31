//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi

package ww

/*

	shims.go contains shim functions for WASM imports, which allows
	symbol names to resolve on non-WASM architectures.

*/

func test(a, b uint32) uint32 {
	return a + b
}
