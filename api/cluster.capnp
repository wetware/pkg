using Go = import "/go.capnp";

@0xfcf6ac08e448a6ac;

$Go.package("cluster");
$Go.import("github.com/wetware/ww/internal/api/cluster");

using PeerID = Text;


interface Anchor {
    ls @0 (path :List(Text)) -> (children :List(Child));
    struct Child {
        name @0 :Text;
        anchor @1 :Anchor;
    }

    walk @1 (path :List(Text)) -> (anchor :Anchor);
}

interface Host extends(Anchor) {
    join @0 (peers :List(AddrInfo)) -> ();
    struct AddrInfo {
        id  @0 :PeerID;
        addrs @1 :List(Data);
    }
}

interface Container extends(Anchor){
    get @0 () -> (data :Data);
    set @1 (data :Data) -> ();
}

interface View {
    iter @0 (handler :Sender) -> ();
    lookup @1 (peerID :PeerID) -> (record :Record, ok :Bool);
 
    using Sender = import "channel.capnp".Sender;
 
    struct Record {
        peer @0 :PeerID;
        ttl  @1 :Int64;
        seq  @2 :UInt64;
    }
}
