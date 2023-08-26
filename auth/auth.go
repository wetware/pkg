package auth

import (
	"context"

	"capnproto.org/go/capnp/v3"
)

type Session[T ~capnp.ClientKind] interface {
	Client() T
	Close() error
}

type Policy[T ~capnp.ClientKind] func(context.Context, T) Session[T]

func (allow Policy[T]) Login(ctx context.Context, t T) Session[T] {
	return allow(ctx, t)
}

func AllowAll[T ~capnp.ClientKind](ctx context.Context, t T) Session[T] {
	return maybe[T]{t} // just(t)
}

func DenyAll[T ~capnp.ClientKind]() Session[T] {
	return maybe[T]{} // nothing
}

type maybe[T ~capnp.ClientKind] struct {
	T T
}

func (t maybe[T]) Client() T {
	return t.T
}

func (t maybe[T]) Close() error {
	capnp.Client(t.T).Release()
	return nil
}
