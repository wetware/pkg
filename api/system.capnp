using Go = import "/go.capnp";

@0x8861ec5893db2e82;

$Go.package("system");
$Go.import("github.com/wetware/pkg/api/system");


struct Event {
    union {
        poll @0  :Void;
        rpc  @1 :import "rpc.capnp".Message;
    }
}


interface Socket {
    notify @0 (event :Event) -> stream;
}