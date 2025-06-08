package csp_server

import "capnproto.org/go/capnp/v3"

var (
	// TODO mikel: this is NOT an elegant or scalable solution to releases.
	// We should map client releases to their cloned counterparts and release if
	// the clone is released.
	PendingReleases = make([]capnp.ReleaseFunc, 0)
)
