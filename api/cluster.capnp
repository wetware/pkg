using Go = import "/go.capnp";

@0xfcf6ac08e448a6ac;

$Go.package("cluster");
$Go.import("github.com/wetware/ww/internal/api/cluster");



struct Record {
    peer @0 :Text;
    ttl @1 :Int64;
    seq @2 :UInt64;
}

struct Iteration {
    record @0 :Record;
    dedadline @1 :Int64;
}

interface Cluster {
    iter @0 (handler :Handler, bufSize :Int32) -> ();
    lookup @1 (peerID :Text) -> (record :Record, ok :Bool);
    interface Handler {
        handle @0 (iterations :List(Iteration)) -> ();
    }
}