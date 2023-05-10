using Go = import "/go.capnp";

@0xfcf6ac08e448a6ac;

$Go.package("cluster");
$Go.import("github.com/wetware/ww/internal/api/cluster");


interface Host {
    # Host represents a physical or virtual machine instance
    # participating in the cluster.

    view @0 () -> (view :Capability);
    # View returns the host's partial view of the cluster. A
    # view represents a pointin-time snapshot of the cluster,
    # and makes no guarantee of consistency.
    #
    # The returned :Capability SHALL be a CASM :View type.

    pubSub @1 () -> (pubSub :import "pubsub.capnp".Router);
    # PubSub returns an interface to the host's pubsub overlay.
    # Callers can use this to connect to arbitrary topics.
    #
    # Note that the PubSub capability confers the ability to join
    # any topic that can be designated by name. Attempts to limit
    # access to topics based on name amounts to ambient authority,
    # and therefore strongly discouraged. A better approach is to
    # wrap PubSub in a capability that resolves sturdy references
    # to Topic capabilities.

    root @2 () -> (root :import "anchor.capnp".Anchor);
    # Root returns the host's root Anchor, which confers access to
    # all shared memory on the host.

    debug @3 () -> (debugger :Capability);
    # Debugger provides a set of tools for debugging live Wetware
    # hosts.  The debugger can potentially reveal sensitive data,
    # including cryptographic secrets and SHOULD NOT be provided
    # to untrusted parties.

    registry @4 () -> (registry :import "registry.capnp".Registry);
    # Registry returns a Service Registry capability, which is used for 
    # discovering and providing service. This way, applications can find each other.

    executor @5 () -> (executor :import "process.capnp".Executor);
    # Executor provides a way of spawning and running WASM-based
    # processes.
}
