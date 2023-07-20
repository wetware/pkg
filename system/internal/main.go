//go:generate tinygo build -o main.wasm -target=wasi -scheduler=asyncify main.go

package main

import (
	"fmt"
	"unsafe"
)

var hostbuf, guestbuf []byte

// main is just a demo right now.  It is the counterpart to
// the net.Conn that is the subject of ww.go's demo.  It reads
// a greeting from the pipe, and prints a reply.
//
// This proves we can stream bytes bidirectionally between host
// and guest.  From here, we should hopefully be able to wrap it
// in an rpc.Conn and have it "just work".
func main() {
	fmt.Println("Hello, Wetware!")
	// // DEMO
	// buf := make([]byte, 42)
	// n, err := pipe{}.Read(buf)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }

	// fmt.Println(string(buf[:n]))
	// pipe{}.Write([]byte("nm, u?"))
	// // -- DEMO
}

//go:export __init
func wwinit(host, guest uint32) {
	hostbuf = make([]byte, int(host))
	guestbuf = make([]byte, int(guest))
	initBuffers(bytesToPointer(hostbuf), bytesToPointer(guestbuf))
}

//go:wasm-module ww
//go:export __init_buffers
func initBuffers(guestBufPtr, hostBufPtr uintptr)

// //go:wasm-module ww
// //go:export _recv
// func recv(offset uintptr, size uint32) uint64

// //go:wasm-module ww
// //go:export _send
// func send(offset uintptr, size uint32) uint64

// // pipe is a wrapper around send/recv/kill that satisfies the
// // io.ReadWriteCloser interface.
// //
// // Implementation involves low-level WASM bit-twiddling.  It
// // is intentionally kept simple.
// type pipe struct{}

// func (pipe) Close() error {
// 	if errno := kill(); errno != 0 {
// 		return fmt.Errorf("%d", errno) // TODO:  parse errno
// 	}

// 	return nil
// }

// func (pipe) Read(b []byte) (int, error) {
// 	pointer := bytesToPointer(b)
// 	return ioResult(recv(pointer, uint32(len(b))))
// }

// func (pipe) Write(b []byte) (int, error) {
// 	pointer := bytesToPointer(b)
// 	return ioResult(send(pointer, uint32(len(b))))

// }

// func ioResult(u64 uint64) (n int, err error) {
// 	n = int(u64 >> 32)
// 	if uint32(u64) != 0 {
// 		err = fmt.Errorf("%d", uint32(u64)) // TODO:  parse errno
// 	}

// 	return
// }

//go:inline
func bytesToPointer(b []byte) uintptr {
	pointer := unsafe.SliceData(b)
	return uintptr(unsafe.Pointer(pointer))
}

//go:inline
func stringToPointer(s string) uintptr {
	pointer := unsafe.StringData(s)
	return uintptr(unsafe.Pointer(pointer))
}
