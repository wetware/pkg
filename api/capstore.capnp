using Go = import "/go.capnp";

@0xbbe22aa2756d2943;

$Go.package("capstore");
$Go.import("github.com/wetware/pkg/api/capstore");


interface CapStore {
    # CapStore works as a capability storage mapping strings to capabilities.
    set @0 (id :Text, cap :Capability) -> ();
    get @1 (id :Text) -> (cap :Capability);
}
