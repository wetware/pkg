package client

import (
	"context"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	ww "github.com/lthibault/wetware/pkg"
)

/*
	Contains the client anchor implementation.  client.Client serves as a root anchor,
	which lazily connects to sub-anchors (i.e. remote hosts).
*/

type anchor struct {
	id   peer.ID
	host host.Host
}

func (a anchor) Ls() ww.Iterator {
	panic("function NOT IMPLEMENTED")
}

func (a anchor) Walk(ctx context.Context, path []string) ww.Anchor {
	panic("function NOT IMPLEMENTED")
}
