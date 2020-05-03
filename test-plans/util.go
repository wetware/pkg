package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/wetware/pkg/discover"
	"github.com/testground/sdk-go/sync"
)

/*
	util.go contains implementations of native ww interfaces for use with Testground.
*/

var topic = sync.NewTopic("discover", new(peer.AddrInfo))

// SyncProto implements discovery over github.com/testground/sdk-go/sync.
// It does not close the underlying sync.Client.
type SyncProto struct {
	Client *sync.Client
	N      int // number of peers
}

// DiscoverPeers over Testground sync service.
func (d SyncProto) DiscoverPeers(ctx context.Context) ([]*peer.AddrInfo, error) {
	ch := make(chan interface{}, 1)
	defer close(ch)

	sub, err := d.Client.Subscribe(ctx, topic, ch)
	if err != nil {
		return nil, err
	}

	addrs := make([]*peer.AddrInfo, 0, d.N)
	for {
		select {
		case v := <-ch:
			addrs = append(addrs, v.(*peer.AddrInfo))
		case <-sub.Done():
			rand.Shuffle(d.N, func(i, j int) {
				addrs[i], addrs[j] = addrs[j], addrs[i]
			})
			return addrs[:2], nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

}

// Start advertising the service in the background.  Does not block.
// Subsequent calls to Start MUST be preceeded by a call to Close.
func (d SyncProto) Start(s discover.Service) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	as, err := s.Network().InterfaceListenAddresses()
	if err != nil {
		return err
	}

	if _, err = d.Client.Publish(ctx, topic, &peer.AddrInfo{
		ID:    s.ID(),
		Addrs: as,
	}); err != nil {
		return err
	}

	return nil
}

// Close stops the active service advertisement.  Once called, Start can be called
// again.
func (d SyncProto) Close() error {
	return nil
}
