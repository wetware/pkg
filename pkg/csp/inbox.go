package csp

import (
	"errors"

	capnp "capnproto.org/go/capnp/v3"
)

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
	}
	return clients, nil
}
