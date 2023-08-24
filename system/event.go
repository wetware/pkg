package system

import "sync/atomic"

type eventfd struct {
	Ctr atomic.Uint32
}
