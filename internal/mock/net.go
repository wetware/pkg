//go:generate mockgen -source=net.go -destination=net/net.go -package=mock_net

package mock

import (
	"net"
)

type (
	PacketConn interface{ net.PacketConn }
)
