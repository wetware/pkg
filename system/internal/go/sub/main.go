//go:generate env GOOS=wasip1 GOARCH=wasm gotip build -o main.wasm main.go
package main

import (
	"os"
)

const EXIT_CODE = 42

func main() {
	os.Exit(EXIT_CODE)
}
