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
	case <-ctx.Done():
	}

	// The future may have resolved due to a canceled context, in which
	// case there is a race-condition in the above select.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return f.Err()
}

type FuturePtr struct{ *capnp.Future }

func (f FuturePtr) Client() *capnp.Client {
	ptr, err := f.Ptr()
	if err != nil {
		return capnp.ErrorClient(err)
	}

	return ptr.Interface().Client()
}

func (f FuturePtr) Await(ctx context.Context) (capnp.Ptr, error) {
	select {
	case <-f.Done():
	case <-ctx.Done():
	}

	// The future may have resolved due to a canceled context, in which
	// case there is a race-condition in the above select.
	if ctx.Err() != nil {
		return capnp.Ptr{}, ctx.Err()
	}

	return f.Ptr()
}

func (f FuturePtr) AwaitBytes(ctx context.Context) ([]byte, error) {
	ptr, err := f.Await(ctx)
	return ptr.Data(), err
}

func (f FuturePtr) AwaitString(ctx context.Context) (string, error) {
	ptr, err := f.Await(ctx)
	return ptr.Text(), err
}

func (f FuturePtr) Bytes() ([]byte, error) {
	ptr, err := f.Ptr()
	return ptr.Data(), err
}

func (f FuturePtr) String() (string, error) {
	ptr, err := f.Ptr()
	return ptr.Text(), err
}

func (f FuturePtr) Ptr() (capnp.Ptr, error) {
	s, err := f.Struct()
	if err != nil {
		return capnp.Ptr{}, err
	}

	return s.Ptr(0)
}
