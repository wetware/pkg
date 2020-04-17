using Go = import "/go.capnp";
@0xb3f8acfcffafd8e8;
$Go.package("api");
$Go.import("github.com/lthibault/wetware/internal/api");

struct Heartbeat {
	id @0 :Text;
    ttl @1 :Int64;
	addrs @2 :List(Data);
}

struct GoAway {
	id @0 :Text;
}
