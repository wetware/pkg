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
        char @2 :Int32;
        str @3 :Text;
        keyword @4 :Text;
        symbol @5 :Text;
        path @6 :Text;
        list @7 :LinkedList;
        vector @8 :Vector;
        # map @9 :CHAMP;
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
    
    # Using Text paths saves a couple of bytes since we don't have to wrap the text,
    # which is effectively a List(uint8), in _another_ list.
    walk @1 (path :Text) -> (anchor :Anchor);
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

