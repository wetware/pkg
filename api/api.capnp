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
        bool @1 :Bool;
        i64 @2 :Int64;
        bigInt @3 :Data;
        f64 @4 :Float64;
        bigFloat @5 :Text;
        frac @6 :Frac;
        char @7 :Int32;
        str @8 :Text;
        keyword @9 :Text;
        symbol @10 :Text;
        path @11 :Text;
        list @12 :LinkedList;
        vector @13 :Vector;
        # map @14 :CHAMP;
        proc @14 :Proc;
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

