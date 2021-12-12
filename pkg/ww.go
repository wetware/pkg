package ww

import (
	"github.com/libp2p/go-libp2p-core/protocol"
)

const (
	Version             = "0.0.0"
	Proto   protocol.ID = "/" + Version
)

// var (
// 	// A wetware invocation is an iota of time.
// 	ww = make(chan func(context.Context) error)

// 	runtime Registers
// )

// type Registers struct {
// 	Clock syncutil.Ctr64 `ww:"tick"`
// 	Load  syncutil.Ctr64 `ww:"uint64"`
// }

// func LoadRegisters() *Registers {
// 	ptr := unsafe.Pointer(&runtime)
// 	return (*Registers)(atomic.LoadPointer(&ptr))
// }

// func CompareAndSwap(old, new *Registers) bool {
// 	ptr := unsafe.Pointer(&runtime)
// 	return atomic.CompareAndSwapPointer(&ptr,
// 		unsafe.Pointer(old),
// 		unsafe.Pointer(new))
// }

// func init() {
// 	go func() {
// 		for {
// 			ww <- func(ctx context.Context) error {
// 				// Error handlers run before time is incremented.
// 				//
// 				// Formally:  ctx.Err() happens before runtime.Clock.Incr()
// 				// ""
// 				defer runtime.Clock.Incr()

// 				return Next(ctx)
// 			}
// 		}
// 	}()
// }

// func Next(ctx context.Context) error {
// 	select {
// 	case state := <-ww:
// 		return state(bind(ctx))

// 	case <-ctx.Done():
// 		return ctx.Err()
// 	}
// }

// func bind(ctx context.Context) context.Context {
// 	ctx = version(ctx)
// 	ctx = proto(ctx)
// 	ctx = state(ctx)
// 	return ctx
// }

// func version(ctx context.Context) context.Context {
// 	return context.WithValue(ctx, versionKey(Version), Version)
// }

// func proto(ctx context.Context) context.Context {
// 	return context.WithValue(ctx, protoKey(Version), Version)
// }

// func state(ctx context.Context) context.Context {
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	runtime.Load.Incr()
// 	defer runtime.Load.Decr()

// 	return context.WithValue(ctx, Cancel, cancel)
// }

// type (
// 	key        string
// 	protoKey   key
// 	versionKey key
// 	cancelKey  key
// )

// /* add package-level keys here */
// const Cancel = cancelKey("")
