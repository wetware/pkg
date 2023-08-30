package auth

import (
	"context"
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/crypto"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/record"
	api "github.com/wetware/pkg/api/cluster"
)

type Nonce [16]byte

func (Nonce) Domain() string {
	return "ww.auth"
}

func (Nonce) Codec() []byte {
	return []byte{0xde, 0xea} // TODO:  pick a good value for this
}

func (n Nonce) MarshalRecord() ([]byte, error) {
	return n[:], nil
}

func (n *Nonce) UnmarshalRecord(buf []byte) error {
	if size := copy((*n)[:], buf); size != 16 {
		return fmt.Errorf("invalid nonce size: %d", size)
	}

	return nil
}

type Signer func(*Nonce) (*record.Envelope, error)

func AccountFromPrivKey[T ~capnp.ClientKind](pk crypto.PrivKey) Signer {
	return func(n *Nonce) (*record.Envelope, error) {
		return record.Seal(n, pk)
	}
}

func AccountFromHost[T ~capnp.ClientKind](h local.Host) Signer {
	privKey := h.Peerstore().PrivKey(h.ID())
	return AccountFromPrivKey[T](privKey)
}

// Sign([]byte) (*record.Envelope, error)
func (sign Signer) Client() capnp.Client {
	if sign == nil {
		return capnp.Client{}
	}

	client := api.Signer_ServerToClient(&signOnce{sign: sign})
	return capnp.Client(client)
}

func (sign Signer) Account() api.Signer {
	return api.Signer(sign.Client())
}

type signOnce struct {
	called bool
	sign   Signer
}

func (once *signOnce) Sign(ctx context.Context, call api.Signer_sign) error {
	if once.called {
		return errors.New("signer already used")
	}
	once.called = true

	challenge, err := call.Args().Challenge()
	if err != nil {
		return err
	}

	var n Nonce
	if size := copy(n[:], challenge); size != 16 {
		return fmt.Errorf("invalid nonce size: %d", size)
	}

	// return empty bytes; most callers will fail, but root-level access
	// can be implemented as a policy that allows any signer that doesn't
	// return an exception.
	if once.sign == nil {
		return nil
	}

	e, err := once.sign(&n)
	if err != nil {
		return err
	}

	signed, err := e.Marshal()
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetSigned(signed)
}
