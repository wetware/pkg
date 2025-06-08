//go:generate env GOOS=wasip1 GOARCH=wasm go build -o bootstrap.wasm bootstrap.go
package main

import (
	"context"
	"fmt"
	"os"

	ww "github.com/wetware/pkg/guest/system"
)

func main() {
	ctx := context.Background()

	caps, releases, err := ww.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		for _, release := range releases {
			release()
		}
	}()

	if len(caps) == 0 {
		fmt.Println("No capabilities found in bootstrap")
		os.Exit(1)
	}

	session, err := ww.Login(ctx, caps[0])
	if err != nil {
		panic(err)
	}

	fmt.Printf("Successfully bootstrapped session %v\n", session)
	processes, release, err := session.Exec().Ps(ctx)
	if err != nil {
		panic(err)
	}
	defer release()

	selfFound := false
	pid := ww.Pid()
	for _, proc := range processes {
		if proc.Pid() == pid {
			selfFound = true
			break
		}
	}
	if !selfFound {
		fmt.Println("Self process not found in the list of processes")
		os.Exit(1)
	}
	fmt.Println("Self found successfully")
}
