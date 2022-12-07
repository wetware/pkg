using Go = import "/go.capnp";

@0xfcba4f486a351ac3;

$Go.package("discovery");
$Go.import("github.com/wetware/ww/internal/api/discovery");


interface DiscoveryService {
    provider @0 (name :Text) -> (provider :Provider);
    locator @1 (name :Text) -> (locator :Locator);
}

interface Provider {
    provide @0 (addrs :Addr) -> ();
}

interface Locator {
    findProviders @0 (chan :Sender(Addr)) -> ();
    
    using Sender = import "channel.capnp".Sender;
}

struct Addr {
    maddrs   @0 :List(Multiaddr);

    using Multiaddr = Data;
}

struct Message {
    union {
        request @0 :Request;  # TODO: use Void type
        response @1 :Response;
    }

    struct Request {}

    struct Response {
        addr @0 :Addr;
    }
}