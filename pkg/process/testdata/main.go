package main

import "os"

/*
	build with:  tinygo build -o pkg/process/testdata/main.wasm -target=wasi -scheduler=none pkg/process/testdata/main.go
*/

//export run
func run() {
	os.Exit(99)
}

func main() {}
