package csp

import (
	"context"
	"errors"

	capnp "capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/api/process"
)

type BootContext api.BootContext

// Open extracts the contents of the BootContext as a list of capnp.Clients
func (i BootContext) Open(ctx context.Context) ([]capnp.Client, error) {
	bootContext := api.BootContext(i)
	of, _ := bootContext.Open(context.TODO(), nil)

	<-of.Done()
	or, err := of.Struct()
	if err != nil {
		return nil, err
	}

	list, err := or.Content()
	if err != nil {
		return nil, err
	}

	return ListToClients(list)
}

// ClientsToNewList encodes a list of capnp.Clients into a new capnp.PointerList
func ClientsToNewList(caps ...capnp.Client) (capnp.PointerList, error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return capnp.PointerList{}, err
	}
	l, err := capnp.NewPointerList(seg, int32(len(caps)))
	if err != nil {
		return capnp.PointerList{}, err
	}

	return l, ClientsToExistingList(&l, caps...)
}

// ClientsToExistingList encodes a list of capnp.Clients into an existing capnp.PointerList
func ClientsToExistingList(pl *capnp.PointerList, caps ...capnp.Client) error {
	l := *pl
	for i, cap := range caps {
		if !cap.IsValid() {
			return errors.New("invalid client when converting to list")
		}
		_, iSeg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return err
		}
		l.Set(i, cap.AddRef().EncodeAsPtr(iSeg))
	}
	return nil
}

// ListToClients decodes a capnp.PointerList to a list of capnp.Clients
func ListToClients(list capnp.PointerList) ([]capnp.Client, error) {
	clients := make([]capnp.Client, list.Len())

	for i := 0; i < list.Len(); i++ {
		clientPtr, err := list.At(i)
		if err != nil {
			return clients, err
		}
		var client capnp.Client
		client = client.DecodeFromPtr(clientPtr)
		if !client.IsValid() {
			return clients, errors.New("could not decode client from pointer")
		}
		clients[i] = client
	}
	return clients, nil
}
