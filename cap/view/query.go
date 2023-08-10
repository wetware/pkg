package view

import (
	"errors"
	"fmt"

	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p/core/peer"
	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cluster/routing"
)

/*
	Selectors
*/

func All() Selector {
	return func(s SelectorStruct) error {
		s.SetAll()
		return nil
	}
}

func Match(index routing.Index) Selector {
	return func(s SelectorStruct) error {
		return bindIndex(s.NewMatch, index)
	}
}

func From(index routing.Index) Selector {
	return func(s SelectorStruct) error {
		return bindIndex(s.NewFrom, index)
	}
}

/*
	Helpers
*/

func bindIndex(fn func() (api.View_Index, error), index routing.Index) error {
	target, err := fn()
	if err != nil {
		return err
	}

	target.SetPrefix(index.Prefix())

	switch index.String() {
	case "id", "peer":
		return bindPeer(target, index)

	case "server":
		return bindServer(target, index)

	case "host":
		return bindHost(target, index)

	case "meta":
		return bindMeta(target, index)
	}

	return fmt.Errorf("invalid index: %s", index)
}

func bindPeer(target api.View_Index, index routing.Index) error {
	switch ix := index.(type) {
	case routing.PeerIndex:
		b, err := ix.PeerBytes()
		if err == nil {
			return target.SetPeer(string(b)) // TODO:  unsafe.Pointer
		}
		return err

	case interface{ Peer() peer.ID }:
		return target.SetPeer(ix.Peer().String())
	}

	return errors.New("not a peer index")
}

func bindServer(target api.View_Index, index routing.Index) error {
	switch ix := index.(type) {
	case routing.ServerIndex:
		b, err := ix.ServerBytes()
		if err == nil {
			return target.SetServer(b)
		}
		return err

	case interface{ Server() routing.ID }:
		index, err := ix.Server().MarshalText()
		if err == nil {
			defer pool.Put(index)
			err = target.SetServer(index) // copies index
		}
		return err
	}

	return errors.New("not a peer index")
}

func bindHost(target api.View_Index, index routing.Index) error {
	switch ix := index.(type) {
	case routing.HostIndex:
		b, err := ix.HostBytes()
		if err == nil {
			return target.SetHost(string(b)) // TODO:  unsafe.Pointer
		}
		return err

	case interface{ Host() (string, error) }:
		id, err := ix.Host()
		if err == nil {
			err = target.SetHost(id)
		}
		return err
	}

	return errors.New("not a peer index")
}

func bindMeta(target api.View_Index, index routing.Index) error {
	switch ix := index.(type) {
	case routing.MetaIndex:
		b, err := ix.MetaBytes()
		if err == nil {
			err = target.SetMeta(string(b))
		}
		return err

	case interface {
		MetaField() (routing.MetaField, error)
	}:
		f, err := ix.MetaField()
		if err == nil {
			err = target.SetMeta(f.String())
		}
		return err
	}

	return errors.New("not a metadata index")
}
