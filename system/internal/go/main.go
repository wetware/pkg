//go:generate env GOOS=wasip1 GOARCH=wasm gotip build -o main.wasm main.go
package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/ww/api/process"
	csp "github.com/wetware/ww/pkg/csp/client"
)

//go:embed sub/main.wasm
var subProcessBC []byte

const EXIT_CODE = 42

func main() {
	if err := doRpc(); err != nil {
		panic(err)
	}
}

func doRpc() error {
	fd := 3 // pre-opened tcp conn listener
	f := os.NewFile(uintptr(fd), "")

	if err := syscall.SetNonblock(fd, false); err != nil {
		return err
	}

	defer f.Close()

	l, err := net.FileListener(f)
	if err != nil {
		return err
	}
	defer l.Close()

	tcpConn, err := l.Accept()
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		ErrorReporter: errLogger{},
	})
	defer conn.Close()

	client := conn.Bootstrap(context.Background())
	inbox := process.Inbox(client)
	defer inbox.Release()
	fmt.Println(inbox)

	if err := client.Resolve(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println(inbox)

	clients, err := csp.Inbox(inbox).Open(context.TODO())
	if err != nil {
		panic(err)
	}

	executor := process.Executor(clients[0])

	exec, release := executor.Exec(context.TODO(), func(e process.Executor_exec_Params) error {
		return e.SetBytecode(subProcessBC)
	})
	defer release()
	<-exec.Done()

	proc := exec.Process()
	wait, release := proc.Wait(context.TODO(), nil)
	defer release()
	<-wait.Done()
	waitResult, err := wait.Struct()
	if err != nil {
		return err
	}
	exitCode := waitResult.ExitCode()
	if exitCode != EXIT_CODE {
		return fmt.Errorf("wait: expected '%d' got '%d'", EXIT_CODE, exitCode)
	} else {
		fmt.Println("wait: matched")
	}
	/* */

	return nil
}

type errLogger struct{}

func (e errLogger) ReportError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
