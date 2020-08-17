using Go = import "/go.capnp";

#
# anchor.capnp contains definitions for the Anchor protocol.
#

@0xe7dd644ba93cb72c;

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
        vector @7 :Vector;
        # list @8 :PList;
        # map @9 :CHAMP;
    }
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

