package ww

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-kbucket/peerdiversity"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type DHT interface {
	// Bootstrap allows callers to hint to the routing system to get into a
	// Boostrapped state and remain there.
	Bootstrap(ctx context.Context) error

	// GetRoutingTableDiversityStats fetches the Routing Table Diversity Stats.
	GetRoutingTableDiversityStats() []peerdiversity.CplDiversityStats

	// Provide adds the given cid to the content routing system.
	Provide(ctx context.Context, key cid.Cid, announce bool) error

	// FindProvidersAsync searches for peers who are able to provide a given key
	FindProvidersAsync(ctx context.Context, key cid.Cid, count int) <-chan peer.AddrInfo

	// FindPeer searches for a peer with given ID
	// Note: with signed peer records, we can change this to short circuit once either DHT returns.
	FindPeer(ctx context.Context, pid peer.ID) (peer.AddrInfo, error)

	// PutValue adds value corresponding to given Key.
	PutValue(ctx context.Context, key string, val []byte, opts ...routing.Option) error

	// GetValue searches for the value corresponding to given Key.
	GetValue(ctx context.Context, key string, opts ...routing.Option) ([]byte, error)

	// SearchValue searches for better values from this value
	SearchValue(ctx context.Context, key string, opts ...routing.Option) (<-chan []byte, error)

	// GetPublicKey returns the public key for the given peer.
	GetPublicKey(ctx context.Context, pid peer.ID) (crypto.PubKey, error)
}

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
	ListPeers(topic string) []peer.ID
	BlacklistPeer(pid peer.ID)
	RegisterTopicValidator(topic string, val interface{}, opts ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(topic string) error
}
