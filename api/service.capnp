using Go = import "/go.capnp";

@0xfcba4f486a351ac3;

$Go.package("service");
$Go.import("github.com/wetware/ww/internal/api/service");


interface Registry {
    provide @0 (topic :import "pubsub.capnp".Topic, location :SignedLocation) -> ();
    
    findProviders @1 (topic :import "pubsub.capnp".Topic, chan :Sender(SignedLocation)) -> ();
    using Sender = import "channel.capnp".Sender;
}

struct SignedLocation {
    signature @0 :Data;
    location  @1 :Location;
}

struct Location {
    service @0 :Text;
    id @1 :PeerID;
    union{
        maddrs @2 :List(Multiaddr);
        anchor @3 :AnchorPath;
        custom @4 :AnyPointer;
    }
    meta     @5 :List(Text);

    using PeerID = Text;
    using Multiaddr = Data;
    using AnchorPath = Text;
}

struct Message {
    union {
        request @0 :Void;
        response @1 :SignedLocation;
    }
}
