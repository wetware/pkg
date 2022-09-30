using Go = import "/go.capnp";

@0xfcba4f486a351ac3;

$Go.package("discovery");
$Go.import("github.com/wetware/ww/internal/api/discovery");

interface Discovery {
    advertise @0 (name :Text, addrs :List(Text)) -> (holder :Holder);
    findPeers @1 (name :Text) -> (addrs :List(Text));

    interface Holder{}
}
