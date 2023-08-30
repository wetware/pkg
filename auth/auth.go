package auth

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/record"
)

type Session[T ~capnp.ClientKind] struct {
	Client T
}

func (sess Session[T]) Close() error {
	capnp.Client(sess.Client).Release()
	return nil
}

func (sess Session[T]) Authenticate(ctx context.Context, account Signer[T]) Session[T] {
	return sess
}

type Signer[T ~capnp.ClientKind] func([]byte) (*record.Envelope, error)

// Sign([]byte) (*record.Envelope, error)
func (sign Signer[T]) Client() capnp.Client {
	if sign == nil {
		return capnp.Client{}
	}

	// TODO:  api.Signer_ServerToClient(...)
	panic("NOT IMPLEMENTED")
}

type Policy[T ~capnp.ClientKind] func(context.Context, Signer[T]) Session[T]

func (auth Policy[T]) Authenticate(ctx context.Context, account Signer[T]) Session[T] {
	return auth(ctx, account)
}

func AllowAll[T ~capnp.ClientKind](ctx context.Context, t T) Session[T] {
	return Session[T]{t} // just(t)
}

func DenyAll[T ~capnp.ClientKind]() Session[T] {
	return Session[T]{} // nothing
}

func Failf[T ~capnp.ClientKind](format string, args ...any) Session[T] {
	return Fail[T](fmt.Errorf(format, args...))
}

func Fail[T ~capnp.ClientKind](err error) Session[T] {
	return Session[T]{T(capnp.ErrorClient(err))}
}
