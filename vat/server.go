package vat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"capnproto.org/go/capnp/v3"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/view"
)

// Server provides the Host capability.
type Server struct {
	NS     string
	Host   local.Host
	Auth   auth.Policy
	OnJoin interface {
		Emit(any) error
	}

	Root    auth.Session
	Cluster interface {
		View() view.View
	}

	once sync.Once
	ch   chan network.Stream
}

func (svr *Server) setup() {
	svr.once.Do(func() {
		svr.ch = make(chan network.Stream)
	})
}

// Close the vat.Network implementation.  Note that Close()
// does not affect ongoing requests, nor does it release any
// capabilities.
func (svr *Server) Close() error {
	svr.setup()
	close(svr.ch)
	return nil
}

func (svr *Server) Export() capnp.Client {
	return capnp.NewClient(core.Terminal_NewServer(svr))
}

func (svr *Server) Login(ctx context.Context, call core.Terminal_login) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	res, err := call.AllocResults()
	if err != nil {
		return fmt.Errorf("alloc results: %w", err)
	}

	account, err := auth.Negotiate(ctx, call.Args().Account())
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	// svr.Auth is responsible for incrementing the refcount on the
	// root session.
	return svr.Auth(ctx, res, svr.Root, account)
}
