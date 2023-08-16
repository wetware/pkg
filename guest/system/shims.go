//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi
// +build !purego,!appengine,!wasm,!tinygo.wasm,!wasi

package system

/*
	This file contains shims for the functions exported by the host
	to the guest.
*/

func pollHost() int32 {
	return 0
}
