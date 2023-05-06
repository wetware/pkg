using Go = import "/go.capnp";

@0xfcba4f486a351ac3;

$Go.package("service");
$Go.import("github.com/wetware/ww/internal/api/service");


using Envelope = Data;

interface Registry {
    provide @0 (topic :import "pubsub.capnp".Topic, envelope :Envelope) -> ();
    
    findProviders @1 (topic :import "pubsub.capnp".Topic, chan :Sender(Envelope)) -> ();
    using Sender = import "channel.capnp".Sender;
}

struct Message {
    union {
        request @0 :Void;
        response @1 :Envelope;
    }
}

struct Location {
    service @0 :Text;
    meta     @1 :List(Text);
    union{
        maddrs @2 :List(Multiaddr);
        anchor @3 :AnchorPath;
        custom @4 :AnyPointer;
    }

    using Multiaddr = Data;
    using AnchorPath = Text;
}