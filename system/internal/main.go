//go:generate tinygo build -o main.wasm -target=wasi -scheduler=asyncify main.go

package main

import "fmt"

func main() {
	fmt.Println("hello, wetware!")
}
