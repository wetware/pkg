using Go = import "/go.capnp";

@0xb3f8acfcffafd8e8;

$Go.package("api");
$Go.import("github.com/wetware/ww/internal/api");

struct Heartbeat $Go.doc("Heartbeat is a liveliness message that is broadcast over pubsub.") {
    ttl @0 :Int64;
}
