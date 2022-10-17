package discovery

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	casm "github.com/wetware/casm/pkg"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/service"
)

type DiscoveryService struct {
	api.DiscoveryService
}

func (c *DiscoveryService) Provider(ctx context.Context, name string) (Provider, capnp.ReleaseFunc) {
	fut, release := c.DiscoveryService.Provider(ctx, func(ps api.DiscoveryService_provider_Params) error {
		return ps.SetName(name)
	})

	return Provider{fut.Provider()}, release
}

type Provider struct {
	api.Provider
}

func (c *Provider) Provide(ctx context.Context, infos []peer.AddrInfo) (casm.Future, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	fut, release := c.Provider.Provide(ctx, func(ps api.Provider_provide_Params) error {
		capInfos, err := ToCapInfoList(infos)
		if err != nil {
			return err
		}
		return ps.SetAddrs(capInfos)
	})

	return casm.Future(fut), func() {
		cancel()
		release()
	}
}

func (c *DiscoveryService) Locator(ctx context.Context, name string) (Locator, capnp.ReleaseFunc) {
	fut, release := c.DiscoveryService.Locator(ctx, func(ps api.DiscoveryService_locator_Params) error {
		return ps.SetName(name)
	})

	return Locator{api.Locator(fut.Locator())}, release
}

type Locator struct {
	api.Locator
}

func (c *Locator) FindProviders(ctx context.Context) (casm.Iterator[peer.AddrInfo], capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	handler := make(handler, 32)

	fut, release := c.Locator.FindProviders(ctx, func(ps api.Locator_findProviders_Params) error {
		return ps.SetChan(chan_api.Sender_ServerToClient(handler))
	})

	iterator := casm.Iterator[peer.AddrInfo]{
		Future: casm.Future(fut),
		Seq:    handler, // TODO: decide buffer size
	}

	return iterator, func() {
		cancel()
		release()
	}
}

type handler chan peer.AddrInfo

func (ch handler) Shutdown() { close(ch) }

func (ch handler) Next() (b peer.AddrInfo, ok bool) {
	b, ok = <-ch
	return
}

func (ch handler) Send(ctx context.Context, call chan_api.Sender_send) error {
	ptr, err := call.Args().Value()
	if err == nil {
		// It's okay to block here, since there is only one writer.
		// Back-pressure will be handled by the BBR flow-limiter.
		info, err := toInfo(api.AddrInfo(ptr.Struct()))
		if err != nil {
			return err
		}

		select {
		case ch <- info:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

func toInfo(capInfo api.AddrInfo) (peer.AddrInfo, error) {
	var info peer.AddrInfo

	id, err := capInfo.Id()
	if err != nil {
		return info, err
	}

	info.ID = peer.ID(id)

	addrs, err := capInfo.Addrs()
	if err != nil {
		return info, err
	}

	maddrs := make([]ma.Multiaddr, 0, addrs.Len())
	for i := 0; i < addrs.Len(); i++ { // TODO: is this the most efficient way?
		data, err := addrs.At(i)
		if err != nil {
			return info, err
		}

		addr, err := ma.NewMultiaddrBytes(data)
		if err != nil {
			return info, err
		}

		maddrs = append(maddrs, addr)
	}

	info.Addrs = maddrs

	return info, nil
}

func ToCapInfoList(infos []peer.AddrInfo) (api.AddrInfo_List, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	capInfos, err := api.NewAddrInfo_List(seg, int32(len(infos)))
	if err != nil {
		return api.AddrInfo_List{}, err
	}

	for i, info := range infos {
		capInfo, err := toCapInfo(info)
		if err != nil {
			return api.AddrInfo_List{}, err
		}

		if err := capInfos.Set(i, capInfo); err != nil {
			return api.AddrInfo_List{}, err
		}
	}
	return capInfos, nil
}

func toCapInfo(info peer.AddrInfo) (api.AddrInfo, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	capInfo, err := api.NewAddrInfo(seg)
	if err != nil {
		return api.AddrInfo{}, err
	}

	if err := capInfo.SetId(string(info.ID)); err != nil {
		return api.AddrInfo{}, err
	}

	capAddrs, err := capInfo.NewAddrs(int32(len(info.Addrs)))
	if err != nil {
		return api.AddrInfo{}, err
	}

	for i, addr := range info.Addrs {
		if err := capAddrs.Set(i, addr.Bytes()); err != nil {
			return api.AddrInfo{}, err
		}
	}

	return capInfo, err
}
