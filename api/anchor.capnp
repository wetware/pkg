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
        anchors @0 :List(SubAnchor);
        struct SubAnchor {
            subpath @0 :Text;
            subanchor @1 :Anchor;
        }
    }

    # Using Text paths saves a couple of bytes since we don't have to wrap the text,
    # which is effectively a List(uint8), in _another_ list.
    walk @1 (path :Text) -> (anchor :Anchor);
}
