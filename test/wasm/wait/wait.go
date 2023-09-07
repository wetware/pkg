//go:generate env GOOS=wasip1 GOARCH=wasm go build -o wait.wasm wait.go
package main

import (
	"time"
)

func main() {
	for {
		time.Sleep(1 * time.Second)
	}
}
