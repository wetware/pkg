using Go = import "/go.capnp";

@0xfcf6ac08e448a6ac;

$Go.package("routing");
$Go.import("github.com/wetware/ww/internal/api/routing");



struct Record {
    peer @0 :Text;
    ttl @1 :Int64;
    seq @2 :UInt64;
}

struct PushedIteration {
    record @0 :Record;
    dedadline @1 :Int64;
}

interface Routing {
    iter @0 (handler :Handler) -> ();
    lookup @1 (peerID :Text) -> (record :Record, ok :Bool);
    interface Handler {
        handle @0 (pi :PushedIteration) -> ();
    }
}