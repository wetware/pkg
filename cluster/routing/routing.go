//go:generate mockgen -source=routing.go -destination=../../../internal/mock/cluster/routing/routing.go -package=mock_routing

package routing

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unsafe"

	"capnproto.org/go/capnp/v3"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ID is an opaque identifier that identifies a unique host instance
// on the network.   A fresh ID is generated for each cluster.Router
// instance, making it possible to distinguish between multiple runs
// of a libp2p host with a fixed peer identity.
//
// IDs are not guaranteed to be globally unique.
type ID uint64

func (id ID) String() string {
	b := id.Bytes()
	defer pool.Put(b)

	buf := pool.Get(16)
	hex.Encode(buf, b)

	return *(*string)(unsafe.Pointer(&buf))
}

func (id ID) Bytes() []byte {
	buf := pool.Get(8)
	binary.BigEndian.PutUint64(buf, uint64(id))
	return buf
}

func (id *ID) UnmarshalText(b []byte) error {
	buf := pool.Get(8)
	defer pool.Put(buf)

	_, err := hex.Decode(buf, b)
	if err == nil {
		*id = ID(binary.BigEndian.Uint64(buf))
	}

	return err
}

func (id ID) MarshalText() ([]byte, error) {
	b := pool.Get(16)
	binary.BigEndian.PutUint64(b, uint64(id))

	buf := id.Bytes()
	defer pool.Put(buf)

	hex.Encode(b, buf)
	return b, nil
}

func (id ID) Loggable() map[string]any {
	return map[string]any{
		"server": id,
	}
}

// Record is an entry in the routing table.
type Record interface {
	Server() ID
	Peer() peer.ID
	Seq() uint64
	TTL() time.Duration
	Host() (string, error)
	Meta() (Meta, error)
}

// Snapshot provides iteration strategies over an isolated snapshot
// of the routing-table.  Implementations MUST NOT mutate the state
// of the routing table, and MUST support concurrent iteration.
type Snapshot interface {
	Get(Index) (Iterator, error)
	GetReverse(Index) (Iterator, error)
	LowerBound(Index) (Iterator, error)
	ReverseLowerBound(Index) (Iterator, error)
}

// Index is a pointer to a 'column' in the routing table's schema.
// Indexes MUST implement index methods corresponding to the index
// name returned by String().  See schema.go for more information.
type Index interface {
	// String returns the index name.
	String() string

	// Prefix returns true if the index is a prefix match
	Prefix() bool
}

// PeerIndex is an optional interface for Index that designates
// the "id" index in the routing table. The Record type MAY also
// implement PeerIndex to provide fast, allocation-free indexing
// of peer IDs.
type PeerIndex interface {
	PeerBytes() ([]byte, error)
}

// ServerIndex is an optional interface for Index that designates
// the "server" index in the routing table.   The Record type MAY
// also implement ServerIndex to provide allocation-free indexing.
type ServerIndex interface {
	ServerBytes() ([]byte, error)
}

// HostIndex is an optional interface for Index that designates
// the "id" index in the routing table. The Record type MAY also
// implement HostIndex to provide fast, allocation-free indexing
// of hostnames.
type HostIndex interface {
	HostBytes() ([]byte, error)
}

// MetaIndex is an optional interface for Index that designates
// a single key-value pair.   Note that Record does NOT support
// this interface, since the Meta type already provides its own
// indexing method.
type MetaIndex interface {
	MetaBytes() ([]byte, error)
}

// Iterator is a stateful object that enumerates routing
// records.  Iterator's methods are NOT guaranteed to be
// thread-safe, but implementations MUST permit multiple
// iterators to operate concurently.
//
// Implementations MAY operate on immutable snapshots of
// routing-table state, so callers SHOULD consume record
// streams promptly.
type Iterator interface {
	// Next pops a record from the head of the stream and
	// returns it to the caller. Subsequent calls to Next
	// will return a different record. When the stream is
	// exhausted, Next returns nil.
	Next() Record
}

// Meta is an indexed set of key-value pairs describing
// arbitrary metadata.
type Meta capnp.TextList

func (m Meta) String() string {
	return capnp.TextList(m).String()
}

// Len returns the number of metadata fields present in
// the set.
func (m Meta) Len() int {
	return capnp.TextList(m).Len()
}

// At returns the metadata field at index i.
func (m Meta) At(i int) (MetaField, error) {
	s, err := capnp.TextList(m).At(i)
	if err != nil {
		return MetaField{}, err
	}

	return ParseField(s)
}

// Get returns the value associated with the supplied key.
// If the key is not found, Get returns ("", nil).  Errors
// are reserved for failures in reading or parsing fields.
func (m Meta) Get(key string) (string, error) {
	for i := 0; i < m.Len(); i++ {
		field, err := m.At(i)
		if err != nil {
			return "", err
		}

		if field.Key == key {
			return field.Value, err
		}
	}

	return "", nil
}

// Index returns a set of indexes for the metadata fields.
func (m Meta) Index() (indexes [][]byte, err error) {
	var index []byte
	for i := 0; i < m.Len(); i++ {
		index, err = capnp.TextList(m).BytesAt(i)
		if err != nil {
			break
		}

		indexes = append(indexes, index)
	}

	return
}

// MetaField is a key-value pair.
type MetaField struct {
	Key, Value string
}

func ParseField(s string) (MetaField, error) {
	switch ss := strings.Split(s, "="); len(ss) {
	case 0:
		return MetaField{}, errors.New("missing key")

	case 1:
		return MetaField{}, errors.New("separator not found")

	default:
		return MetaField{
			Key:   ss[0],
			Value: strings.Join(ss[1:], "="),
		}, nil

	}
}

func (f MetaField) String() (s string) {
	if f.Key != "" {
		s = f.Key + "=" + f.Value
	}

	return
}
