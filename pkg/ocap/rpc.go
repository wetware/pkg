package ocap

import (
	"context"

	"capnproto.org/go/capnp/v3"
)

type Future struct{ *capnp.Future }

func (f Future) Err() error {
	_, err := f.Struct()
	return err
}

func (f Future) Await(ctx context.Context) error {
	select {
	case <-f.Done():
		return f.Err()

	case <-ctx.Done():
		return ctx.Err()
	}
}

type FuturePtr struct{ *capnp.Future }

func (f FuturePtr) Err() error {
	_, err := f.Struct()
	return err
}

func (f FuturePtr) Await(ctx context.Context) (capnp.Ptr, error) {
	select {
	case <-f.Done():
		return f.Ptr()

	case <-ctx.Done():
		return capnp.Ptr{}, ctx.Err()
	}
}

func (f FuturePtr) AwaitBytes(ctx context.Context) ([]byte, error) {
	ptr, err := f.Await(ctx)
	return ptr.Data(), err
}

func (f FuturePtr) AwaitString(ctx context.Context) (string, error) {
	ptr, err := f.Await(ctx)
	return ptr.Text(), err
}

func (f FuturePtr) AwaitClient(ctx context.Context) (*capnp.Client, error) {
	ptr, err := f.Await(ctx)
	return ptr.Interface().Client(), err
}

func (f FuturePtr) Ptr() (capnp.Ptr, error) {
	s, err := f.Struct()
	if err != nil {
		return capnp.Ptr{}, err
	}

	return s.Ptr(0)
}

func (f FuturePtr) Bytes() ([]byte, error) {
	ptr, err := f.Ptr()
	return ptr.Data(), err
}

func (f FuturePtr) String() (string, error) {
	ptr, err := f.Ptr()
	return ptr.Text(), err
}

// Client blocks until the future resolves, and then returns a client.
// If an error is encountered during future resolution, the error is
// wrapped in a client whose methods always fail with the wrapped error.
//
// Callers who wish to inspect the error should call f.Ptr() and then
// convert the pointer to a client via ptr.Interface().Client().
func (f FuturePtr) Client() *capnp.Client {
	ptr, err := f.Ptr()
	if err != nil {
		return capnp.ErrorClient(err)
	}

	return ptr.Interface().Client()
}
