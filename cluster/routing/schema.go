package routing

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/hashicorp/go-memdb"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p/core/peer"
	b58 "github.com/mr-tron/base58/base58"
)

var schema = memdb.TableSchema{
	Name: "record",
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: idIndexer{},
		},
		"server": {
			Name:    "server",
			Unique:  true,
			Indexer: serverIndexer{},
		},
		"ttl": {
			Name:    "ttl",
			Indexer: timeIndexer{},
		},
		"host": {
			Name:         "host",
			AllowMissing: true,
			Indexer:      hostnameIndexer{},
		},
		"meta": {
			Name:         "meta",
			AllowMissing: true,
			Indexer:      metaIndexer{},
		},
	},
}

type idIndexer struct{}

func (idIndexer) FromObject(obj any) (bool, []byte, error) {
	switch r := obj.(type) {
	case PeerIndex:
		index, err := r.PeerBytes()
		return err == nil, index, err

	case Record:
		return true, peerToBytes(r.Peer()), nil
	}

	return false, nil, errType(obj)
}

func (idIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != 1 {
		return nil, errNArgs(args)
	}

	switch arg := args[0].(type) {
	case PeerIndex:
		return arg.PeerBytes()

	case Record:
		return peerToBytes(arg.Peer()), nil

	case peer.ID:
		return peerToBytes(arg), nil

	case string:
		return stringToBytes(arg), nil

	}

	return nil, errType(args[0])
}

func (idIndexer) PrefixFromArgs(args ...any) ([]byte, error) {
	return idIndexer{}.FromArgs(args...)
}

func peerToBytes(id peer.ID) []byte {
	index := b58.Encode(*(*[]byte)(unsafe.Pointer(&id)))
	return stringToBytes(index)
}

type serverIndexer struct{}

func (serverIndexer) FromObject(obj any) (bool, []byte, error) {
	switch rec := obj.(type) {
	case ServerIndex:
		index, err := rec.ServerBytes()
		return err == nil, index, err

	case Record:
		index, err := rec.Server().MarshalText()
		return true, index, err
	}

	return false, nil, errType(obj)
}

func (serverIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != 1 {
		return nil, errNArgs(args)
	}

	switch arg := args[0].(type) {
	case ServerIndex:
		return arg.ServerBytes()

	case Record:
		return arg.Server().MarshalText()

	case ID:
		return arg.MarshalText()

	case string:
		return stringToBytes(arg), nil

	case []byte:
		return arg, nil
	}

	return nil, errType(args[0])
}

func (serverIndexer) PrefixFromArgs(args ...any) ([]byte, error) {
	return serverIndexer{}.FromArgs(args...)
}

type timeIndexer struct{}

func (timeIndexer) FromObject(obj any) (bool, []byte, error) {
	if r, ok := obj.(*record); ok {
		return true, timeToBytes(r.Deadline), nil
	}

	return false, nil, errType(obj)
}

func (timeIndexer) FromArgs(args ...any) ([]byte, error) {
	t, err := argsToTime(args...)
	if err != nil {
		return nil, err
	}

	return timeToBytes(t), nil
}

func timeToBytes(t time.Time) []byte {
	ms := t.UnixNano()
	buf := pool.Get(8)

	// big-endian; avoids branching in radix tree
	binary.BigEndian.PutUint64(buf, uint64(ms))

	return buf
}

func argsToTime(args ...any) (time.Time, error) {
	if len(args) != 1 {
		return time.Time{}, errNArgs(args)
	}

	if t, ok := args[0].(time.Time); ok {
		return t, nil
	}

	return time.Time{}, errType(args[0])
}

type hostnameIndexer struct{}

func (hostnameIndexer) FromObject(obj any) (bool, []byte, error) {
	switch rec := obj.(type) {
	case HostIndex:
		index, err := rec.HostBytes()
		return len(index) != 0, index, err

	case Record:
		name, err := rec.Host()
		if err != nil || name == "" {
			return false, nil, err
		}

		return true, stringToBytes(name), nil
	}

	return false, nil, errType(obj)
}

func (hostnameIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != 1 {
		return nil, errNArgs(args)
	}

	switch arg := args[0].(type) {
	case HostIndex:
		return arg.HostBytes()

	case Record:
		name, err := arg.Host()
		if err != nil || name == "" {
			return nil, err
		}

		return stringToBytes(name), nil

	case string:
		return stringToBytes(arg), nil
	}

	return nil, errType(args[0])
}

func (hostnameIndexer) PrefixFromArgs(args ...any) ([]byte, error) {
	return hostnameIndexer{}.FromArgs(args...)
}

type metaIndexer struct{}

func (metaIndexer) FromObject(obj any) (bool, [][]byte, error) {
	if r, ok := obj.(Record); ok {
		meta, err := r.Meta()
		if err != nil || meta.Len() == 0 {
			return false, nil, err
		}

		indexes, err := meta.Index()
		return true, indexes, err
	}

	return false, nil, errType(obj)
}

func (metaIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != 1 {
		return nil, errNArgs(args)
	}

	switch arg := args[0].(type) {
	case MetaIndex:
		return arg.MetaBytes()

	case string:
		return stringToBytes(arg), nil
	}

	return nil, errType(args[0])
}

func (metaIndexer) PrefixFromArgs(args ...any) ([]byte, error) {
	return metaIndexer{}.FromArgs(args...)
}

func stringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

func errType(v any) error {
	return fmt.Errorf("invalid type: %s", reflect.TypeOf(v))
}

func errNArgs(args []any) error {
	return fmt.Errorf("expected one argument (got %d)", len(args))
}
