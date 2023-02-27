package main

import (
	"io"
	"os"
)

var _ping = []byte("Ping\n")

//export echo
func echo() {
	io.Copy(io.MultiWriter(os.Stdout, os.Stderr), os.Stdin)
}

//export echoErr
func echoErr() {
	io.Copy(os.Stderr, os.Stdin)
}

//export echoOut
func echoOut() {
	io.Copy(os.Stdout, os.Stdin)
}

//export ping
func ping() {
	io.MultiWriter(os.Stdout, os.Stderr).Write(_ping)
}

//export pingErr
func pingErr() {
	os.Stdout.Write(_ping)
}

//export pingOut
func pingOut() {
	os.Stdout.Write(_ping)
}

func main() {

}
