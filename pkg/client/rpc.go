package client

import (
	"context"
	"fmt"
	"io"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/lthibault/log"
	protoutil "github.com/wetware/casm/pkg/util/proto"
	rpcutil "github.com/wetware/ww/internal/util/rpc"
	ww "github.com/wetware/ww/pkg"
)

type RPCFactory interface {
	New(ctx context.Context, ns string, h host.Host, d discovery.Discoverer) (*rpc.Conn, error)
}

type BasicRPCFactory struct {
	Log                log.Logger        `optional:"true"`
	ErrorReporter      rpc.ErrorReporter `optional:"true"`
	DisableCompression bool              `optional:"true" name:"capnp-no-packed"`
	Options            rpc.Options
}

func (f BasicRPCFactory) New(ctx context.Context, ns string, h host.Host, d discovery.Discoverer) (*rpc.Conn, error) {
	if f.Log == nil {
		f.Log = log.New()
	}

	if f.ErrorReporter == nil {
		f.ErrorReporter = rpcutil.ErrReporterFunc(func(err error) {
			f.Log.WithError(err).Warn("rpc error")
		})
	}

	info, err := f.discover(ctx, ns, d)
	if err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}

	if err := h.Connect(ctx, info); err != nil {
		return nil, fmt.Errorf("connect (%s): %w", info.ID.ShortString(), err)
	}

	s, err := h.NewStream(ctx, info.ID, f.proto(ns))
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}

	return rpc.NewConn(f.newTransport(s), f.options(ns)), nil
}

func (f BasicRPCFactory) discover(ctx context.Context, ns string, d discovery.Discoverer) (info peer.AddrInfo, err error) {
	var (
		ch <-chan peer.AddrInfo
		ok bool
	)

	ch, err = d.FindPeers(ctx, ns /*discovery.Limit(1)*/)
	if err == nil {
		select {
		case info, ok = <-ch:
			if !ok {
				err = fmt.Errorf("no peers")
			}
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	return
}

func (f BasicRPCFactory) newTransport(rwc io.ReadWriteCloser) rpc.Transport {
	if f.DisableCompression {
		return rpc.NewStreamTransport(rwc)
	}

	return rpc.NewPackedStreamTransport(rwc)
}

func (f BasicRPCFactory) proto(ns string) protocol.ID {
	id := protoutil.AppendStrings(ww.Proto, ns)
	if !f.DisableCompression {
		id = protoutil.AppendStrings(id, "packed")
	}

	return id
}

func (f BasicRPCFactory) options(ns string) *rpc.Options {
	if f.Options.ErrorReporter == nil {
		var log = f.Log.WithField("ns", ns)
		f.Options.ErrorReporter = rpcutil.ErrReporterFunc(func(err error) {
			log.WithError(err).Warn("rpc error")
		})
	}

	return &f.Options
}
