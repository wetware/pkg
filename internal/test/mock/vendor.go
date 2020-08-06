//go:generate mockgen -package mock_vendor -destination ../../../internal/test/mock/vendor/mock_vendor.go github.com/wetware/ww/internal/test/mock Host,Discovery

package mocktest

import (
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
)

// Host interface
type Host interface {
	host.Host
}

// Discovery interface
type Discovery interface {
	discovery.Discovery
}
