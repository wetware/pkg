package auth

import (
	"context"
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/crypto"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/record"
	cluster_api "github.com/wetware/pkg/api/cluster"
	core_api "github.com/wetware/pkg/api/core"
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

func SignerFromHost(h local.Host) Signer {
	privKey := h.Peerstore().PrivKey(h.ID())
	return SignerFromPrivKey(privKey)
}

func SignerFromPrivKey(privKey crypto.PrivKey) Signer {
	return func(n *Nonce) (*record.Envelope, error) {
		return record.Seal(n, privKey)
	}
}

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

	client := cluster_api.Signer_ServerToClient(&signOnce{sign: sign})
	return capnp.Client(client)
}

func (sign Signer) Account() cluster_api.Signer {
	return cluster_api.Signer(sign.Client())
}

func (sign Signer) Bind(ctx context.Context) func(core_api.Terminal_login_Params) error {
	return func(call core_api.Terminal_login_Params) error {
		return call.SetAccount(cluster_api.Signer(sign.Client()))
	}
}

type signOnce struct {
	called bool
	sign   Signer
}

func (once *signOnce) Sign(ctx context.Context, call cluster_api.Signer_sign) error {
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
