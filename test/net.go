//go:generate mockgen -source=net.go -destination=net/net.go -package=test_net

package test

import (
	"net"
)

type (
	PacketConn interface{ net.PacketConn }
)
