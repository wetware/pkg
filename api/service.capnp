using Go = import "/go.capnp";

@0xfcba4f486a351ac3;

$Go.package("service");
$Go.import("github.com/wetware/ww/internal/api/service");


interface DiscoveryService {
    provider @0 (name :Text) -> (provider :Provider);
    locator @1 (name :Text) -> (locator :Locator);
}

interface Provider {
    provide @0 (addrs :List(AddrInfo)) -> ();
}

interface Locator {
    findProviders @0 (chan :Sender(AddrInfo)) -> ();
    
    using Sender = import "channel.capnp".Sender;
}

struct AddrInfo {
    id      @0 :PeerID;
    addrs   @1 :List(Multiaddr);

    using PeerID   = Text;
    using Multiaddr = Data;
}

struct Message {
    union {
        request @0 :Request;  # TODO: use Void type
        response @1 :Response;
    }

    struct Request {}

    struct Response {
        addrs @0 :List(AddrInfo);
    }
}