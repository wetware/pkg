using Go = import "/go.capnp";

@0xc60caf5632fce538;

$Go.package("anchor");
$Go.import("github.com/wetware/ww/internal/api/anchor");


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

    using PeerID = import "cluster.capnp".PeerID;
}

interface Container extends(Anchor){
    get @0 () -> (data :Data);
    set @1 (data :Data) -> ();
}