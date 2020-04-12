package ww

import (
	"io"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	iface "github.com/ipfs/interface-go-ipfs-core"
	host "github.com/libp2p/go-libp2p-core/host"
)

// New Host
func New(node *core.IpfsNode) (Host, error) {
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil, err
	}

	return p2pHost{
		CoreAPI: api,
		Host:    node.PeerHost,
		Closer:  node,
	}, nil
}

type p2pHost struct {
	iface.CoreAPI
	host.Host
	io.Closer
}

func (h p2pHost) Close() error      { return h.Closer.Close() }
func (h p2pHost) Stream() StreamAPI { return h.Host }
