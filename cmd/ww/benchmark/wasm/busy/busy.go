//go:generate env CGO_ENABLED=0 CGO= GOOS=wasip1 GOARCH=wasm go build -o busy.wasm busy.go
package main

import (
	"context"
	"runtime"
	"strconv"

	ww "github.com/wetware/pkg/guest/system"
)

const (
	TOTAL_CYCLES = iota
	YIELD_CYCLES
)

func main() {
	ctx := context.Background()

	_, err := ww.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}

	total, err := strconv.ParseInt(ww.Args()[TOTAL_CYCLES], 10, 64)
	if err != nil {
		panic(err)
	}
	yield, err := strconv.ParseInt(ww.Args()[YIELD_CYCLES], 10, 64)
	if err != nil {
		panic(err)
	}

	var x int64
	for i := int64(0); i < total; i++ {
		x++
		if yield != 0 && x%yield == 0 {
			// this affecst the Wazero runtime, no the Go runtime (?)
			runtime.Gosched()
		}
	}
}
