package system

import (
	"context"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero"
	"golang.org/x/exp/slog"
)

// Declare the host module from a set of exported functions.
var HostModule wazergo.HostModule[*Host] = functions{
	"__poll": wazergo.F0((*Host).Poll),
}

func Instantiate(ctx context.Context, r wazero.Runtime, sock *Socket) (
	*wazergo.ModuleInstance[*Host],
	context.Context,
	error,
) {
	instance, err := wazergo.Instantiate(ctx, r, HostModule, WithSocket(sock))
	ctx = wazergo.WithModuleInstance(ctx, instance)
	return instance, ctx, err
}

// The `functions` type impements `HostModule[*Socket]`, providing the
// module name, map of exported functions, and the ability to create instances
// of the module type.
type functions wazergo.Functions[*Host]

func (f functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Host] {
	return (wazergo.Functions[*Host])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*Host, error) {
	mod := &Host{
		// ...
	}
	wazergo.Configure(mod, opts...)
	return mod, nil
}

type Option = wazergo.Option[*Host]

func WithSocket(sock *Socket) Option {
	return wazergo.OptionFunc(func(host *Host) {
		host.sock = sock
	})
}

type Host struct {
	sock *Socket
}

func (host *Host) Close(ctx context.Context) error {
	return host.sock.Close()
}

func (host *Host) Poll(ctx context.Context) types.Error {
	slog.Info("host.Poll called by guest")
	return types.OK
}
