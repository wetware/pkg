package system

import (
	"context"
	"io"
	"net"
	"os"
	"syscall"

	"capnproto.org/go/capnp/v3/rpc"
)

func load() file {
	return file{os.NewFile(uintptr(PREOPENED_FD), "")}
}

func stream(sock io.ReadWriteCloser) rpc.Transport {
	return rpc.NewStreamTransport(sock)
}

type file struct{ *os.File }

func (file) Network() string  { return "" }
func (f file) String() string { return f.Name() }
func (f file) FD() int        { return int(f.File.Fd()) }

type fileDialer struct{}

func (fileDialer) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	return dial(addr.(file).File)
}

// dial pre-opened file descriptor
func dial(f *os.File) (net.Conn, error) {
	if err := syscall.SetNonblock(int(f.Fd()), false); err != nil {
		return nil, err
	}

	l, err := net.FileListener(f)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	return l.Accept()
}
