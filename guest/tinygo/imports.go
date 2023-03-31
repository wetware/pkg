//go:build wasm || tinygo.wasm || wasi

package ww

//go:wasm-module ww
//go:export __test
func test(a, b uint32) uint32
