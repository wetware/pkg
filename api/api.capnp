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

    struct ProcSpec {
        struct Goroutine {
            value @0 :Value;
        }

        struct OSProc {
            args @0 :List(Text);
            env @1 :List(EnvVar);
            dir @2 :Text;
        }

        struct Docker {
        }

        struct EnvVar {
            name @0 :Text;
            value @1 :Text;
        }

        union{
            goroutine @0 :Goroutine;
            osProc @1 :OSProc;
            docker @2 :Docker;
        }
    }
    
    # Using Text paths saves a couple of bytes since we don't have to wrap the text,
    # which is effectively a List(uint8), in _another_ list.
    walk @1 (path :Text) -> (anchor :Anchor);
    load @2 () -> (value :Value);
    store @3 (value :Value) -> ();
    go @4 (spec :ProcSpec) -> ();  # TODO:  return a Process interface.
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

