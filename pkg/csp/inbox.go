package csp

import (
	"context"
	"errors"

	capnp "capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/api/process"
)

type Inbox api.Inbox

// Open extracts the contents of the Inbox as a list of capnp.Clients
func (i Inbox) Open(ctx context.Context) ([]capnp.Client, error) {
	inbox := api.Inbox(i)
	of, _ := inbox.Open(context.TODO(), nil)

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

// ClientsToList encodes a list of capnp.Clients into a capnp.PointerList
func ClientsToList(caps ...capnp.Client) (capnp.PointerList, error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return capnp.PointerList{}, err
	}
	l, err := capnp.NewPointerList(seg, int32(len(caps)))
	if err != nil {
		return capnp.PointerList{}, err
	}
	for i, cap := range caps {
		_, iSeg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return capnp.PointerList{}, err
		}
		l.Set(i, cap.EncodeAsPtr(iSeg))
	}
	return l, nil
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
