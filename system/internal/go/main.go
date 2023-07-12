//go:generate env GOOS=wasip1 GOARCH=wasm gotip build -o main.wasm main.go
package main

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/tetratelabs/wazero/sys"
	"github.com/wetware/ww/api/process"
	csp "github.com/wetware/ww/pkg/csp"
	ww "github.com/wetware/ww/wasm"
)

//go:embed sub/main.wasm
var subProcessBC []byte

const EXIT_CODE = 42

func main() {
	ctx := context.Background()
	if err := doRpc(ctx); err != nil {
		panic(err)
	}
}

func doRpc(ctx context.Context) error {

	clients, closers, err := ww.Init(ctx)
	defer closers.Close()
	if err != nil {
		panic(err)
	}

	executor := process.Executor(clients[0])

	proc, release := csp.Executor(executor).Exec(ctx, subProcessBC)
	defer release()
	proc.Wait(ctx)
	err = proc.Wait(ctx)

	ee, ok := err.(*sys.ExitError)
	if !ok {
		return err
	}
	exitCode := ee.ExitCode()

	if exitCode != EXIT_CODE {
		return fmt.Errorf("wait: expected '%d' got '%d'", EXIT_CODE, exitCode)
	} else {
		fmt.Println("wait: matched")
	}

	return nil
}

type errLogger struct{}

func (e errLogger) ReportError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
