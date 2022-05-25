package anchor

import (
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/wetware/ww/internal/api/anchor"
)

func bindHostInfo(host anchor.Host_AddrInfo, info peer.AddrInfo) error {
	if err := host.SetId(string(info.ID)); err != nil {
		return err
	}

	addrs, err := host.NewAddrs(int32(len(info.Addrs)))
	if err != nil {
		return err
	}

	for i, addr := range info.Addrs {
		if err = addrs.Set(i, addr.Bytes()); err != nil {
			break
		}
	}

	return err
}

func bindAddrInfo(info *peer.AddrInfo, host anchor.Host_AddrInfo) error {
	s, err := host.Id()
	if err != nil {
		return err
	}

	if info.ID, err = peer.IDFromString(s); err != nil {
		return err
	}

	addrs, err := host.Addrs()
	if err != nil {
		return err
	}

	for i := 0; i < addrs.Len(); i++ {
		b, err := addrs.At(i)
		if err != nil {
			return err
		}

		m, err := ma.NewMultiaddrBytes(b)
		if err != nil {
			return err
		}

		info.Addrs = append(info.Addrs, m)
	}

	return nil
}
