package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

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

	// Join a PubSub topic.
	topic := cluster.Join(ctx, "pubsub-example")

	// Publish a message periodically in a separate goroutine.
	go func() {
		msg := fmt.Sprintf("hello from %s", host.ID())
		for {
			err := topic.Publish(ctx, []byte(msg))
			if err != nil {
				log.Fatal(err)
			}

			time.Sleep(time.Second)
		}
	}()

	// Subscribe to the topic, so that we can receive updates.
	sub, err := topic.Subscribe(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Cancel()

	// Print messages received on the subscription.  This will
	// include messages we have sent ourselves.
	for {
		b, err := sub.Next(ctx)
		if err != nil {
			break
		}

		log.Println(string(b))
	}
}
