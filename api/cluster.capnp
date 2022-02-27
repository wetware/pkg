using Go = import "/go.capnp";

@0xfcf6ac08e448a6ac;

$Go.package("cluster");
$Go.import("github.com/wetware/ww/internal/api/cluster");


interface Anchor {
    ls @0 (path :List(Text), handler :Handler) -> ();
    walk @1 (path :List(Text)) -> (anchor :Anchor);

    interface Handler {
        handle @0 (anchors :List(AnchorElement)) -> ();

        struct AnchorElement{
            name @0 :Text;
            anchor @1 :Anchor;
        }
    }
}

interface Host extends(Anchor) {
    host @0 () -> (host :Text);
}

interface Container extends(Anchor){
    
}

interface View {
    iter @0 (handler :Handler) -> ();
    lookup @1 (peerID :Text) -> (record :Record, ok :Bool);
 
    interface Handler {
        handle @0 (records :List(Record)) -> ();
    }
 
    struct Record {
        peer @0 :Text;
        ttl @1 :Int64;
        seq @2 :UInt64;
    }
}
