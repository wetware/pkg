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
}
