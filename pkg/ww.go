package ww

import (
	"github.com/libp2p/go-libp2p-core/protocol"

	protoutil "github.com/wetware/casm/pkg/util/proto"
)

const (
	Version             = "0.0.0"
	Proto   protocol.ID = "/ww/" + Version
)

var match = protoutil.Match(
	protoutil.Prefix("ww"),
	protoutil.SemVer(Version))

// Subprotocol returns a protocol.ID that matches the
// pattern:  /ww/<version>/<ns>/<...>
func Subprotocol(ns string, ss ...string) protocol.ID {
	return protoutil.AppendStrings(Proto,
		append([]string{ns}, ss...)...)
}

// NewMatcher returns a stream matcher for a protocol.ID
// that matches the pattern:  /ww/<version>/<ns>
func NewMatcher(ns string) protoutil.MatchFunc {
	return match.Then(protoutil.Exactly(ns))
}
