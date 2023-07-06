using Go = import "/go.capnp";

@0xefb5a91f96d44de3;

$Go.package("anchor");
$Go.import("github.com/wetware/ww/internal/api/anchor");


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

    cell @2 () -> (loader :Loader, storer :Storer, swapper :Swapper);
    # Cell returns a set of capabilities that provide access to a boxed
    # value, assigned to the Anchor. Zero or more of these capabilities
    # MAY be witheld, in which case the corresponding return value will
    # be null. The loader and storer capabilities respectively map onto
    # read and write permissions. The swapper is an extended read/write
    # capability, which SHALL be witheld if either loader or storer are
    # witheld.

    struct Child {
        anchor @0 :Anchor;
        name   @1 :Text;
    }

    struct Value {
        # Value is a union type that can be assigned to an Anchor.
        
        union {
            null @0 :Void;
            # Null value indicates that the Anchor is empty, i.e. it
            # contains no value.

            chan    :union {
            # Chan values contain some sort of channel. The union is
            # used as a type hint on the receiving side.

                closer     @1 :import "channel.capnp".Closer;
                sender     @2 :import "channel.capnp".Sender;
                recver     @3 :import "channel.capnp".Recver;
                sendCloser @4 :import "channel.capnp".SendCloser;
                chan       @5 :import "channel.capnp".Chan;
            }

            # proc    :group {  # TODO
            # }
        }
    }

    interface Loader {
        # Loader is a read-only interface to a value.   It grants the
        # bearer the authority to access an Anchor's underlying value
        # without modifying it.

        load @0 () -> (value :Value);
        # Load the Anchor's value atomically.  Note that a concurrent
        # thread may change the Anchor's value at any point before or
        # after a call to load.   To conditionally modify an Anchor's
        # value, use Swapper.CompareAndSwap.
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
        # Store().  Use CompareAndSwap to perform conditional stores
        # atomically.
    }

    interface Swapper {
        # Swapper is an extended read-write interface to an Anchor's
        # value. It provides methods for replacing values atomically.
        # Methods of Swapper guarantee that no intermediate state is
        # observed by concurrent threads throughout the operation.

        swap  @0 (new :Value) -> (old :Value);
        # Swap replaces the Anchor's current value with the new value,
        # and returns the value that was replaced.

        compareAndSwap @1 (old :Value, new :Value) -> (swapped :Bool);
        # CompareAndSwap tests whether the Anchor's current value is
        # equal to old, and performs a Swap() operation if it is.
    }
}
