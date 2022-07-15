package main

import (
	"context"
	"fmt"
	"io"
	"log"

	bootutil "github.com/wetware/casm/pkg/boot/util"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"
)

var ctx = context.Background()

func main() {
	// Instantiate the libp2p host that the Wetware client will use.
	host, err := vat.NewClientHost()
	if err != nil {
		log.Fatal(err)
	}
	defer host.Close()

	// Create a bootstrap service, capable of locating active peers
	// in the cluster. To keep things simple, we will use multicast
	// UDP over the loopback interface. In rare cases, you MAY need
	// to change 'lo0' to some other value.
	const addr = "/ip4/228.8.8.8/udp/8822/quic/multicast/lo0"

	// Dial into the network interface to obtain a bootstrapper.
	boot, err := bootutil.DialString(host, addr)
	if err != nil {
		log.Fatal(err)
	}
	defer boot.(io.Closer).Close()

	// Instantiate the dialer.  This will locate a cluster node,
	// connect to it, and return a *client.Node.
	dialer := client.Dialer{
		// This represents the client's Cap'n Proto vat, which is
		// analogous to a 'Host' in libp2p parlance.
		Vat: vat.Network{
			NS:   "ww", // the default cluster namespace
			Host: host,
		},

		// Create a new multicast surveyor and assign it to the
		// bootstrap discovery service.
		Boot: boot,
	}

	// Dial into the cluster.  This returns a *client.Node, which
	// is the main client interface to the Wetware cluster.
	cluster, err := dialer.Dial(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cluster.Close()

	log.Printf("joined cluster")

	// List hosts in the cluster
	it := cluster.Ls(ctx)
	for it.Next() {
		path := it.Anchor().Path()
		fmt.Println(path)
	}

	if it.Err() != nil {
		log.Fatal(err)
	}
}
