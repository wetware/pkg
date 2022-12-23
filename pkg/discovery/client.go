package discovery

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	ma "github.com/multiformats/go-multiaddr"
	casm "github.com/wetware/casm/pkg"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/discovery"
)

type Addr struct {
	Maddrs []ma.Multiaddr
	// TODO: metada or any other field
}

func (addr Addr) String() string {
	return fmt.Sprintf("%v", addr.Maddrs)
}

type DiscoveryService api.DiscoveryService

func (c DiscoveryService) Provider(ctx context.Context, name string) (Provider, capnp.ReleaseFunc) {
	fut, release := api.DiscoveryService(c).Provider(ctx, func(ps api.DiscoveryService_provider_Params) error {
		return ps.SetName(name)
	})

	return Provider(fut.Provider()), release
}

func (c DiscoveryService) Release() {
	api.DiscoveryService(c).Release()
}

type Provider api.Provider

func (c Provider) Provide(ctx context.Context, addr Addr) (casm.Future, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	fut, release := api.Provider(c).Provide(ctx, func(ps api.Provider_provide_Params) error {
		capAddr, err := ToCapAddr(addr)
		if err != nil {
			return err
		}
		return ps.SetAddrs(capAddr)
	})

	return casm.Future(fut), func() {
		cancel()
		release()
	}
}

func (c Provider) Release() {
	api.Provider(c).Release()
}

func (c DiscoveryService) Locator(ctx context.Context, name string) (Locator, capnp.ReleaseFunc) {
	fut, release := api.DiscoveryService(c).Locator(ctx, func(ps api.DiscoveryService_locator_Params) error {
		return ps.SetName(name)
	})

	return Locator(fut.Locator()), release
}

type Locator api.Locator

func (c Locator) Release() {
	api.Locator(c).Release()
}

func (c Locator) FindProviders(ctx context.Context) (casm.Iterator[Addr], capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	handler := make(handler, 32)

	fut, release := api.Locator(c).FindProviders(ctx, func(ps api.Locator_findProviders_Params) error {
		return ps.SetChan(chan_api.Sender_ServerToClient(handler))
	})

	iterator := casm.Iterator[Addr]{
		Future: casm.Future(fut),
		Seq:    handler, // TODO: decide buffer size
	}

	return iterator, func() {
		cancel()
		release()
	}
}

type handler chan Addr

func (ch handler) Shutdown() { close(ch) }

func (ch handler) Next() (b Addr, ok bool) {
	b, ok = <-ch
	return
}

func (ch handler) Send(ctx context.Context, call chan_api.Sender_send) error {
	ptr, err := call.Args().Value()
	if err == nil {
		info, err := toAddr(api.Addr(ptr.Struct()))
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

func toAddr(capInfo api.Addr) (Addr, error) {
	addrs, err := capInfo.Maddrs()
	if err != nil {
		return Addr{}, err
	}

	addr := Addr{}
	maddrs := make([]ma.Multiaddr, 0, addrs.Len())
	for i := 0; i < addrs.Len(); i++ { // TODO: is this the most efficient way?
		data, err := addrs.At(i)
		if err != nil {
			return addr, err
		}

		b := make([]byte, len(data))
		copy(b, data)

		maddr, err := ma.NewMultiaddrBytes(b)
		if err != nil {
			return addr, err
		}

		maddrs = append(maddrs, maddr)
	}

	addr.Maddrs = maddrs
	return addr, nil
}

func ToCapAddr(addr Addr) (api.Addr, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	capAddr, err := api.NewAddr(seg)
	if err != nil {
		return api.Addr{}, err
	}

	capMaddrs, err := capAddr.NewMaddrs(int32(len(addr.Maddrs)))
	if err != nil {
		return api.Addr{}, err
	}

	for i, maddr := range addr.Maddrs {
		if err := capMaddrs.Set(i, maddr.Bytes()); err != nil {
			return api.Addr{}, err
		}
	}

	return capAddr, nil
}
