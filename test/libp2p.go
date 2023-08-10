//go:generate mockgen -source=libp2p.go -destination=libp2p/libp2p.go -package=test_libp2p

// Package test contains mockgen targets from external dependencies.
package test

import (
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type (
	Host              interface{ host.Host }
	Conn              interface{ network.Conn }
	Stream            interface{ network.Stream }
	Network           interface{ network.Network }
	Emitter           interface{ event.Emitter }
	CertifiedAddrBook interface{ peerstore.CertifiedAddrBook }
	Discovery         interface{ discovery.Discovery }
)
