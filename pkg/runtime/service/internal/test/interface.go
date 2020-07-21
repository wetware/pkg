package testutil

import (
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	boot "github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime/service"
)

// Host interface
type Host interface {
	host.Host
}

// BootStrategy interface
type BootStrategy interface {
	boot.Strategy
}

// Discovery interface
type Discovery interface {
	discovery.Discovery
}

// Publisher interface
type Publisher interface {
	service.Publisher
}
