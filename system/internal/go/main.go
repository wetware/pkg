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

	of, release := inbox.Open(context.TODO(), nil)
	defer release()

	<-of.Done()
	content := of.Content()
	executor := process.Executor(content)
	/*
		msg := "Hello"
		echo, release := executor.Echo(context.TODO(), func(e process.Executor_echo_Params) error {
			return e.SetA(msg)
		})
		defer release()
		<-echo.Done()
		result, err := echo.Struct()
		if err != nil {
			return err
		}
		b, err := result.B()
		if err != nil {
			return err
		}
		if b != msg {
			return fmt.Errorf("echo: expected '%s' got '%s'", msg, b)
		} else {
			fmt.Println("echo: matched")
		}
	*/

	/* */
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
