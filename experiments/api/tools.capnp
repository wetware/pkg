using Go = import "/go.capnp";

@0x9462f07b5ef19869;

$Go.package("tools");
$Go.import("github.com/wetware/ww/experiments/api/tools");

using Http = import "http.capnp";

interface Tools {
    http @0 () -> (http :Http.Requester);
}