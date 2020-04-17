// Package mautil contains utilities for transforming multiaddrs.
package mautil

import (
	"github.com/multiformats/go-multiaddr"
)

// NewMultiaddrs parses a slice of string-form multiaddrs.
func NewMultiaddrs(addrs ...string) (ms []multiaddr.Multiaddr, err error) {
	ms = make([]multiaddr.Multiaddr, len(addrs))
	for i, a := range addrs {
		if ms[i], err = multiaddr.NewMultiaddr(a); err != nil {
			break
		}
	}

	return
}
