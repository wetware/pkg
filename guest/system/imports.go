//go:build !purego && !appengine && !wasm && !tinygo.wasm && !wasi
// +build !purego,!appengine,!wasm,!tinygo.wasm,!wasi

package system

import (
	"context"

	"github.com/stealthrocket/wazergo/types"
)

func send(offset, length uint32) types.Errno {
	return types.AsErrno(context.Canceled)
}
