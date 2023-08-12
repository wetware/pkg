package host

import (
	"context"

	api "github.com/wetware/pkg/api/cluster"
)

// AuthFailure returns an auth policy that fails with 'err'.
func AuthFailure(err error) AuthPolicy {
	return func(context.Context, api.Host_login) error {
		return err
	}
}

// AuthDenial returns an auth policy that succeeds, but witholds
// all capabilities.
func AuthDenial() AuthPolicy {
	return func(context.Context, api.Host_login) error {
		return nil // RPC succeeds, but no capabilities returned
	}
}

// AuthDisabled returns the supplied policy to all callers.  It is
// equivalent to having no authentication.
func AuthDisabled(sess Session) AuthPolicy {
	return func(ctx context.Context, call api.Host_login) error {
		res, err := call.AllocResults()
		if err != nil {
			return err
		}

		return sess.Bind(res)
	}
}
