package main

import (
	"github.com/wetware/pkg/guest/system"
)

func main() {
	sock := system.Socket()
	defer sock.Close()

	// _, err := io.Copy(sock, strings.NewReader("hello from guest!"))
	// if err != nil {
	// 	panic(err)
	// }
}
