using Go = import "/go.capnp";

#
# anchor.capnp contains definitions for the Anchor protocol.
#

@0xc8aa6d83e0c03a9d;

$Go.package("api");
$Go.import("github.com/lthibault/wetware/internal/api");


# Var is similar to its Clojure homologue (https://clojure.org/reference/vars).
struct Var {
    state @0 :State;
    struct State(V) {
        type @0 :StateType;
        enum StateType {
            proc @0;
            chan @1;
        }

        value @1 :AnyPointer;
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

    get @2 () -> (var :Var);
    set @3 (var :Var) -> ();
}
