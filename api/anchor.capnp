using Go = import "/go.capnp";

@0xefb5a91f96d44de3;

$Go.package("anchor");
$Go.import("github.com/wetware/ww/internal/api/anchor");


interface Anchor {
    ls @0 () -> (children :List(Child));
    struct Child {
        name @0 :Text;
        anchor @1 :Anchor;
    }

    walk @1 (path :Text) -> (anchor :Anchor);
}


interface Host extends(Anchor) {
    join @0 (peers :List(AddrInfo)) -> ();
    
    struct AddrInfo {
        using PeerID = import "cluster.capnp".PeerID;

        id  @0 :PeerID;
        addrs @1 :List(Data);
    }

}


interface Container extends(Anchor){
    get @0 () -> (data :Data);
    set @1 (data :Data) -> ();
}