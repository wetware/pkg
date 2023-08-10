package system

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3"
)

// Boot returns the bootstrap capability for the system.
func Boot[T ~capnp.ClientKind](ctx context.Context) T {
	client := capnp.ErrorClient(errors.New("Boot: NOT IMPLEMENTED"))
	return T(client)
}
