using Go = import "/go.capnp";

@0xbb59054ba43c3861;

$Go.package("http");
$Go.import("github.com/wetware/ww/experiments/api/http");

interface HttpGetter {
    get @0 (url :Text) -> (status :UInt32, body :Data, error :Text);
}

