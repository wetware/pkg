package start

import "github.com/wetware/casm/pkg/cluster/pulse"

/*
 * system.go contains pulse.Hook constructor, which hooks into the OS to provide system
 * config info to the heartbeat system.
 */

// systemHook populates heartbeat messages with system information from the
// operating system.
type systemHook struct{}

func newSystemHook() pulse.Preparer {
	return systemHook{}
}

func (h systemHook) Prepare(pulse.Heartbeat) {
	// TODO:  populate a capnp struct containing metadata for the
	//        local host.  Consider including AWS AR information,
	//        hostname, geolocalization, and a UUID instance id.

	// WARNING:  DO NOT make a syscall each time 'Prepare' is invoked.
	//           Cache results and periodically refresh them.
}
