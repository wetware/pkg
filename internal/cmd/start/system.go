package start

import "github.com/wetware/casm/pkg/cluster/pulse"

/*
 * system.go contains pulse.Hook constructor, which hooks into the OS to provide system
 * config info to the heartbeat system.
 */

func newSystemHook() pulse.Hook {
	return func(h pulse.Heartbeat) {
		// TODO:  populate a capnp struct containing metadata for the
		//        local host.  Consider including AWS AR information,
		//        hostname, geolocalization, and a UUID instance id.
	}
}
