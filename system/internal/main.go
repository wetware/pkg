//go:generate tinygo build -o main.wasm -target=wasi -scheduler=asyncify main.go

package main

import (
	"fmt"
	"os"
	"unsafe"
)

func main() {
	buf := make([]byte, 42)
	n, err := pipe{}.Read(buf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(string(buf[:n]))
	pipe{}.Write([]byte("nm, u?"))

	// 	info(
	// 		stringToPointer("[ LOG ] Hello, Wetware!"),
	// 		uint32(23))

	// buf := make([]byte, 32)
	// u64 := recv(
	//
	//	bytesToPointer(buf),
	//	uint32(len(buf)))
	//
	// fmt.Printf("%064b\n", u64)
	// fmt.Println(string(buf))
}

//go:wasm-module ww
//go:export _info
func info(offset uintptr, size uint32)

//go:wasm-module ww
//go:export _close
func rpcClose() uint32

//go:wasm-module ww
//go:export _recv
func recv(offset uintptr, size uint32) uint64

//go:wasm-module ww
//go:export _send
func send(offset uintptr, size uint32) uint64

type pipe struct{}

func (pipe) Close() error {
	if errno := rpcClose(); errno != 0 {
		return fmt.Errorf("%d", errno) // TODO:  parse errno
	}

	return nil
}

func (pipe) Read(b []byte) (int, error) {
	pointer := bytesToPointer(b)
	return ioResult(recv(pointer, uint32(len(b))))
}

func (pipe) Write(b []byte) (int, error) {
	pointer := bytesToPointer(b)
	return ioResult(send(pointer, uint32(len(b))))

}

func ioResult(u64 uint64) (n int, err error) {
	n = int(u64 >> 32)
	if uint32(u64) != 0 {
		err = fmt.Errorf("%d", uint32(u64)) // TODO:  parse errno
	}

	return
}

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
