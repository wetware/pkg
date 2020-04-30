using Go = import "/go.capnp";

#
# anchor.capnp contains definitions for the Anchor protocol.
#

@0xc8aa6d83e0c03a9d;

$Go.package("api");
$Go.import("github.com/lthibault/wetware/internal/api");


interface Anchor {
    ls @0 () -> (children :AnchorMap);
    struct AnchorMap {
        subAnchors @0 :List(SubAnchor);
        struct SubAnchor {
            path @0 :Text;
            child @1 :Anchor;
        }
    }

    walk @1 (path :Text) -> (anchor :Anchor);
}
