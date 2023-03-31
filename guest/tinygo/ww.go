// Package ww contains Wetware bindings for WASM guest-code.
package ww

func Test(a, b uint32) uint32 {
	return test(a, b)
}
