using Go = import "/go.capnp";

#
# cluster.capnp contains definitions for the heartbeat protocol.
#
@0xb3f8acfcffafd8e8;

$Go.package("api");
$Go.import("github.com/lthibault/wetware/internal/api");


struct Heartbeat $Go.doc("Heartbeat is a peer liveliness message that is broadcast over pubsub.") {
	id @0 :Text;
    ttl @1 :Int64;
}
