using Go = import "/go.capnp";

#
# anchor.capnp contains definitions for the Anchor protocol.
#

@0xc8aa6d83e0c03a9d;

$Go.package("api");
$Go.import("github.com/wetware/ww/internal/api");


struct Value {
    union {
        nil @0 :Void;
        native @1 :Text;
        bool @2 :Bool;
        i64 @3 :Int64;
        bigInt @4 :Data;
        f64 @5 :Float64;
        bigFloat @6 :Text;
        frac @7 :Frac;
        char @8 :Int32;
        str @9 :Text;
        keyword @10 :Text;
        symbol @11 :Text;
        path @12 :Text;
        list @13 :LinkedList;
        vector @14 :Vector;
        # map @15 :CHAMP;
        proc @15 :Proc;
    }
}


interface Anchor {
    ls @0 () -> (children :List(SubAnchor));
    struct SubAnchor {
        path @0 :Text;
        union {
            root @1 :Void;
            anchor @2 :Anchor;
        }
    }

    walk @1 (path :Text) -> (anchor :Anchor);
    load @2 () -> (value :Value);
    store @3 (value :Value) -> ();
    go @4 (args :List(Value)) -> (proc :Proc);
}


interface Proc {
    wait @0 () -> ();
}

struct Frac {
    numer @0 :Data;
    denom @1 :Data;
}


struct LinkedList {
    count @0 :UInt32;
    head @1 :Value;  # any
    tail @2 :Value;  # ∈ {nil, list}
}


struct Vector {
    count @0 :UInt32;
    shift @1 :UInt8;
    root @2 :Node;
    tail @3 :List(Value);

    struct Node {
        union {
            branches @0 :List(Node);
            values @1 :List(Value);
        }
    }
}

