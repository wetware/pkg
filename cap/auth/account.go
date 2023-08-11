package auth

import (
	"context"
	"errors"

	api "github.com/wetware/pkg/api/cluster"
)

type Signer api.Signer

func (s Signer) AddRef() Signer {
	return Signer(api.Signer(s).AddRef())
}

func (s Signer) Release() {
	api.Signer(s).Release()
}

func (s Signer) Sign(ctx context.Context, challenge []byte) ([]byte, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}
