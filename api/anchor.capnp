using Go = import "/go.capnp";

@0xefb5a91f96d44de3;

$Go.package("anchor");
$Go.import("github.com/wetware/ww/internal/api/anchor");


interface Anchor {
    # Anchor is a shared memory register.  Anchors form a tree structure
    # similar to a filesyste, with one important constraint: nodes along
    # any given path can only access their children.  They cannot access
    # their parents.  This provides strong isolation properties.

    ls   @0 () -> (children :List(Child));
    # ls returns the Anchor's children along with their names.
    # The path to the i'th child is given by:
    #
    #     parent_path + "/" + names[i].
    #

    walk @1 (path :Text) -> (anchor :Anchor);
    # Walk traverses the anchor hierarchy along the specified path. Any
    # anchors in the path that do not currently exist are created along
    # the way.

    struct Child {
        anchor @0 :Anchor;
        name   @1 :Text;
    }
}
