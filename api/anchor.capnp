using Go = import "/go.capnp";

@0xefb5a91f96d44de3;

$Go.package("anchor");
$Go.import("github.com/wetware/pkg/api/anchor");


struct Value {
    union {
        cluster @0 :import "cluster.capnp".View;
        # Cluster is a global, inconsistent view of the cluster. It
        # provides a snapshot-based interface for synchronizing ops
        # in a distributed environment.

        host    @1 :import "core.capnp".Session;

        anchor  @2 :Anchor;
    }
}


interface Anchor {
    # Anchor is a shared memory register, accessible over the network.
    # Anchors form a tree structure similar to a filesystem, with one
    # important property:  each node can only reference its immediate
    # children, and cannot reference its parents. This provides basic
    # isolation between anchors.

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

    cell @2 () -> (loader :Loader, storer :Storer);
    # Cell returns a set of capabilities that provide access to a boxed
    # value, assigned to the Anchor. Zero or more of these capabilities
    # MAY be witheld, in which case the corresponding return value will
    # be null. The loader and storer capabilities respectively map onto
    # read and write permissions.

    struct Child {
        anchor @0 :Anchor;
        name   @1 :Text;
    }

    using Value = AnyPointer;

    interface Loader {
        # Loader is a read-only interface to a value.   It grants the
        # bearer the authority to access an Anchor's underlying value
        # without modifying it.

        load @0 () -> (value :Value);
        # Load the Anchor's value atomically.  Note that a concurrent
        # thread may change the Anchor's value at any point before or
        # after a call to load.
    }

    interface Storer {
        # Storer is a write-only interface to a value.  It grants the
        # bearer the authority to modify an Anchor's underlying value
        # without accessing its currently-stored value.

        store @0 (value :Value, overwrite :Bool) -> (succeeded :Bool);
        # Store the supplied value in the cell.  The returned bool is
        # set to true if the operation succeeded.  Store fails when a
        # value is already stored in the cell unless overwrite is set
        # to true. To clear a value from the cell, overwrite existing
        # values with null.
        #
        # As with the Loader interface, a concurrent thread may load
        # or store a value at any point before and after the call to
        # Store().
    }
}
