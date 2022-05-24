package cluster

import (
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/wetware/ww/internal/api/cluster"
)

func bindHostInfo(host cluster.Host_AddrInfo, info peer.AddrInfo) error {
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

func bindAddrInfo(info *peer.AddrInfo, host cluster.Host_AddrInfo) error {
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

// func newTypeError(want, got interface{}) error {
// 	return fmt.Errorf("expected %s (got %s)",
// 		reflect.TypeOf(want),
// 		reflect.TypeOf(got))
// }

// func newArgLenError(want, got int) error {
// 	noun := "argument"
// 	if want != 1 {
// 		noun += "s"
// 	}

// 	return fmt.Errorf("expected %d %s (got %d)", want, noun, got)
// }

// func anyToString(v any) (string, error) {
// 	if s, ok := v.(string); ok {
// 		return s, nil
// 	}

// 	return "", newTypeError("", v)
// }

// func anyStringToBytes(v any) (b []byte, err error) {
// 	var s string
// 	if s, err = anyToString(v); s != "" {
// 		b = []byte(s) // TODO(performance):  unsafe cast to avoid alloc
// 	}

// 	return
// }
