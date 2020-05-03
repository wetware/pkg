package main

import (
	"context"
	"fmt"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"

	"github.com/lthibault/wetware/pkg/server"
)

// TestAnnounce verifies that hosts are mutually aware of each others' presence.
func TestAnnounce(runenv *runtime.RunEnv) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	netclient := network.NewClient(client, runenv)
	runenv.RecordMessage("Waiting for network initialization")

	netclient.MustWaitNetworkInitialized(ctx)
	runenv.RecordMessage("Network initilization complete")

	host := server.New(
		server.WithTTL(time.Millisecond*100),
		server.WithDiscover(SyncProto{
			Client: client,
			N:      runenv.TestInstanceCount,
		}),
	)

	if err = host.Start(); err != nil {
		return errors.Wrap(err, "start host")
	}

	select {
	case <-time.After(time.Second):
		// heartbeats should have propagated
	case <-ctx.Done():
		return ctx.Err()
	}

	// tests proper
	switch peers := host.Peers(); {
	case len(peers) != runenv.TestInstanceCount:
		msg := fmt.Sprintf("expected %d peers, found %d", runenv.TestInstanceCount, len(peers))
		runenv.SLogger().Error(msg)
		err = errors.New(msg)
	case !contains(peers, host.ID()):
		msg := fmt.Sprintf("%s not in peer set", host.ID())
		runenv.SLogger().Error(msg)
		err = errors.New(msg)
	}

	return err
}

func contains(ps []peer.ID, p peer.ID) bool {
	set := make(map[peer.ID]struct{})
	for _, px := range ps {
		set[px] = struct{}{}
	}

	_, ok := set[p]
	return ok
}
