package server

/*
	api.go contains the capnp api that is served by the host
*/

const (
	protoID = "/ww/anchor/0.0.0"
)

// func registerAnchor(log logger, host host.Host, rt routingTable) {
// 	host.SetStreamHandler(protoID, func(s network.Stream) {
// 		defer s.Reset()

// 		export := api.Anchor_ServerToClient(rootAnchor{
// 			host:         host,
// 			routingTable: rt,
// 		})

// 		// TODO:  write a stream transport that uses a packed encoder/decoder pair
// 		//
// 		//  Difficulty:  easy.
// 		// 	https: //github.com/capnproto/go-capnproto2/blob/v2.18.0/rpc/transport.go
// 		conn := rpc.NewConn(rpc.StreamTransport(s), rpc.MainInterface(export.Client))

// 		if err := conn.Wait(); err != nil {
// 			log.Log().WithError(err).Error("error in stream handler")
// 		}
// 	})
// }

// type rootAnchor struct {
// 	host         host.Host
// 	routingTable interface{ Contains(peer.ID) bool }
// }

// func (a rootAnchor) Ls(call api.Anchor_ls) error {
// 	return errors.New("NOT IMPLEMENTED")
// 	// cs, err := call.Results.NewChildren()
// 	// if err != nil {
// 	// 	return nil
// 	// }

// 	// peers := a.host.Peerstore().Peers()
// 	// hosts := peers[:0]
// 	// for _, p := range peers {
// 	// 	if a.routingTable.Contains(p) {
// 	// 		hosts = append(hosts, p)
// 	// 	}
// 	// }

// 	// as, err := cs.NewSubAnchors(int32(len(hosts)))
// 	// if err != nil {
// 	// 	return err
// 	// }

// 	// for i, p := range hosts {
// 	// 	a, err := api.NewAnchor_AnchorMap_SubAnchor(nil)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}

// 	// 	if err = a.SetPath(p.String()); err != nil {
// 	// 		return err
// 	// 	}

// 	// 	if err = a.SetChild(newServerAnchor(p)); err != nil {
// 	// 		return err
// 	// 	}

// 	// 	if err = as.Set(i, a); err != nil {
// 	// 		return err
// 	// 	}
// 	// }

// }

// func (a rootAnchor) Walk(call api.Anchor_walk) error {
// 	return errors.New("NOT IMPLEMENTED")
// }
