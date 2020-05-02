using Go = import "/go.capnp";

@0xb3f8acfcffafd8e8;

$Go.package("api");
$Go.import("github.com/lthibault/wetware/internal/api");

struct Heartbeat $Go.doc("Heartbeat is a liveliness message that is broadcast over pubsub.") {
	id @0 :Text;
    ttl @1 :Int64;
}
