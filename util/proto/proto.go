package proto

import (
	"github.com/libp2p/go-libp2p/core/protocol"
)

const Version = "0.1.0"

// Root returns the protocol ID for the supplied namespace.
func Root(ns string) protocol.ID {
	return Join("ww", Version, protocol.ID(ns))
}

// Namespace returns the subprotocol family for the
// supplied namespace.
func Namespace(ns string) []protocol.ID {
	namespace := Root(ns)
	return []protocol.ID{
		namespace + "/packed",
		namespace,
	}
}

// NewMatcher returns a stream matcher for a protocol.ID
// that matches the pattern:  /ww/<version>/<ns>
func NewMatcher(ns string) MatchFunc {
	return Match(
		Exactly("ww"), SemVer(Version), Exactly(ns),
	)
}
