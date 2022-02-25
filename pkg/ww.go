package ww

import (
	"github.com/libp2p/go-libp2p-core/protocol"

	protoutil "github.com/wetware/casm/pkg/util/proto"
	"github.com/wetware/casm/pkg/vat"
)

const (
	Version             = "0.0.0"
	Proto   protocol.ID = "/ww/" + Version
)

var match = vat.NewMatcher("ww").
	Then(protoutil.SemVer(Version))

// Subprotocol returns a protocol.ID that matches the
// pattern:  /casm/<casm-version>/ww/<version>/<ns>/<...>
func Subprotocol(ns string, ss ...string) protocol.ID {
	return vat.Subprotocol("ww", append([]string{Version, ns}, ss...)...)
}

// NewMatcher returns a stream matcher for a protocol.ID
// that matches the pattern:  /ww/<version>/<ns>
func NewMatcher(ns string) protoutil.MatchFunc {
	return match.Then(protoutil.Exactly(ns))
}
