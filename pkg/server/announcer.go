package server

import (
	"context"
	"encoding/binary"
	"time"

	"go.uber.org/fx"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	log "github.com/lthibault/log/pkg"

	ww "github.com/lthibault/wetware/pkg"
)

type announcerConfig struct {
	fx.In

	Ctx context.Context
	Log log.Logger

	Host host.Host

	Namespace string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`

	Topic *pubsub.Topic
}

func announcer(lx fx.Lifecycle, cfg announcerConfig) (err error) {
	a := clusterAnnouner{
		log:    cfg.Log,
		hostID: cfg.Host.ID(),
		ttl:    cfg.TTL,
		mesh:   cfg.Topic,
	}

	ctx, cancel := context.WithCancel(cfg.Ctx)
	lx.Append(fx.Hook{
		// We must wait until the libp2p.Host is listening before
		// announcing ourself to peers.
		OnStart: func(start context.Context) (err error) {
			if err = a.Announce(start); err == nil {
				go a.loop(ctx)
			}

			return
		},
		OnStop: func(stop context.Context) error {
			cancel()
			return nil
		},
	})

	return nil
}

type clusterAnnouner struct {
	log log.Logger

	hostID peer.ID
	ttl    time.Duration

	mesh interface {
		Publish(context.Context, []byte, ...pubsub.PubOpt) error
	}
}

func (a clusterAnnouner) Announce(ctx context.Context) error {
	hb, err := ww.NewHeartbeat(a.hostID, a.ttl)
	if err != nil {
		return err
	}

	b, err := ww.MarshalHeartbeat(hb)
	if err != nil {
		return err
	}

	return a.mesh.Publish(ctx, b)
}

func (a clusterAnnouner) loop(ctx context.Context) {
	ticker := time.NewTicker(a.ttl / 3)
	defer ticker.Stop()

	for range ticker.C {
		switch err := a.Announce(ctx); err {
		case nil:
		case context.Canceled:
			return
		default:
			a.log.WithError(err).Error("failed to announce")
		}
	}
}

func newHeartbeatValidator(ctx context.Context, f filter) pubsub.Validator {
	return func(_ context.Context, pid peer.ID, msg *pubsub.Message) bool {
		hb, err := ww.UnmarshalHeartbeat(msg.GetData())
		if err != nil {
			return false // drop invalid message
		}

		if id := msg.GetFrom(); !f.Upsert(id, seqno(msg), hb.TTL()) {
			return false // drop stale message
		}

		msg.ValidatorData = hb
		return true
	}
}

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}
