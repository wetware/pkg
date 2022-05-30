using Go = import "/go.capnp";

@0xfcf6ac08e448a6ac;

$Go.package("cluster");
$Go.import("github.com/wetware/ww/internal/api/cluster");


using PeerID = Text;


interface View {
    iter   @0 (handler :Sender) -> ();
    lookup @1 (peerID :PeerID) -> (record :Record, ok :Bool);
 
    struct Record {
        peer @0 :PeerID;
        ttl  @1 :Int64;
        seq  @2 :UInt64;
    }
 
    using Sender = import "channel.capnp".Sender;
}


interface Joiner {
    join @0 (peers :List(AddrInfo)) -> ();
}


struct AddrInfo {
    id    @0 :PeerID;
    addrs @1 :List(Data);
}


