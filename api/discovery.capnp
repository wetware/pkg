using Go = import "/go.capnp";

@0xfcba4f486a351ac3;

$Go.package("discovery");
$Go.import("github.com/wetware/ww/internal/api/discovery");


interface DiscoveryService {
    provider @0 (name :Text) -> (provider :Provider);
    locator @1 (name :Text) -> (locator :Locator);
}

interface Provider {
    provide @0 (location :SignedLocation) -> ();
}

interface Locator {
    findProviders @0 (chan :Sender(SignedLocation)) -> ();
    
    using Sender = import "channel.capnp".Sender;
}

struct SignedLocation {
    signature @0: Data;
    location @1: Location;
}


struct Location {
    id @0: PeerID;
    union{
        maddrs   @1 :List(Multiaddr);
        anchor @2 :AnchorPath;
        custom @3 :AnyPointer;
    }
    meta     @4 :List(Text);

    using PeerID = Text;
    using Multiaddr = Data;
    using AnchorPath = Text;
}

struct Message {
    union {
        request @0 :Request;  # TODO: use Void type
        response @1 :Response;
    }

    struct Request {}

    struct Response {
        location @0 :SignedLocation;
    }
}
