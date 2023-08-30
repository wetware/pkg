using Go = import "/go.capnp";

@0xfcf6ac08e448a6ac;

$Go.package("cluster");
$Go.import("github.com/wetware/pkg/api/cluster");


interface Host {
    # Host represents a physical or virtual machine instance
    # participating in the cluster.

    view @0 () -> (view :View);
    # View returns the host's partial view of the cluster. A
    # view represents a pointin-time snapshot of the cluster,
    # and makes no guarantee of consistency.
    #
    # The returned :Capability SHALL be a CASM :View type.

    pubSub @1 () -> (pubSub :import "pubsub.capnp".Router);
    # PubSub returns an interface to the host's pubsub overlay.
    # Callers can use this to connect to arbitrary topics.
    #
    # Note that the PubSub capability confers the ability to join
    # any topic that can be designated by name. Attempts to limit
    # access to topics based on name amounts to ambient authority,
    # and therefore strongly discouraged. A better approach is to
    # wrap PubSub in a capability that resolves sturdy references
    # to Topic capabilities.

    root @2 () -> (root :import "anchor.capnp".Anchor);
    # Root returns the host's root Anchor, which confers access to
    # all shared memory on the host.

    registry @3 () -> (registry :import "registry.capnp".Registry);
    # Registry returns a Service Registry capability, which is used for 
    # discovering and providing service. This way, applications can find each other.

    executor @4 () -> (executor :import "process.capnp".Executor);
    # Executor provides a way of spawning and running WASM-based
    # processes.

    capStore @5 () -> (capStore :import "capstore.capnp".CapStore);
    # CapStore returns a Capability Storage.
}


struct Heartbeat {
    # Heartbeat messages are used to implement an unstructured p2p
    # clustering service.  Hosts periodically emit heartbeats on a
    # pubsub topic (the "namespace") and construct a routing table
    # based on heartbeats received by other peers.
    #
    # Additional metadata can piggyback off of heartbeat messages,
    # allowing indexed operations on the routing table.

    ttl      @0 :Milliseconds;
    # Time-to-live, in milliseconds. The originator is considered
    # failed if a subsequent heartbeat is not received within ttl.
    
    server @1 :UInt64;
    # An opaque identifier that uniquely distinguishes an instance
    # of a host. This identifier is randomly generated each time a
    # host boots.

    host     @2 :Text;
    # The hostname of the underlying host, as reported by the OS.
    # Users MUST NOT assume hostnames to be unique or non-empty.

    meta     @3 :List(Text);
    # A set of optional, arbitrary metadata fields.  Fields are
    # encoded as key-value pairs separated by the '=' rune.  Fields
    # are parsed into keys and values by splitting the string on the
    # first occurrenc of the '=' separator.  Subsequent occurrences
    # are treated as part of the value.

    using Milliseconds = UInt32;
}


interface View {
    # A View is a read-only snapshot of a particular host's routing
    # table. Views are not updated, and should therefore be queried
    # and discarded promptly.

    lookup  @0 (selector :Selector, constraints :List(Constraint)) -> (result :MaybeRecord);
    iter    @1 (handler :Handler, selector :Selector, constraints :List(Constraint)) -> ();
    reverse @2 () -> (view :View);
    
    interface Handler {
        recv @0 (record :Record) -> stream;
    }

    struct Selector {
        union {
            all        @0 :Void;
            match      @1 :Index;
            from       @2 :Index;
        }
    }

    struct Constraint {
        union {
            limit      @0 :UInt64;
            to         @1 :Index;
        }
    }

    struct Index {
        prefix    @0 :Bool;
        union {
            peer   @1 :Text;
            server @2 :Data;
            host   @3 :Text;
            meta   @4 :Text;        # key=value
        }
    }

    struct Record {
        peer      @0 :PeerID;
        server    @1 :UInt64;
        seq       @2 :UInt64;
        heartbeat @3 :Heartbeat;
    }

    struct MaybeRecord {
        union {
            nothing @0 :Void;
            just    @1 :Record;
        }
    }

    using PeerID = Text;
}


interface Terminal {
    login @0 (account :Signer) -> (
        view :import "cluster.capnp".View,
        pubSub :import "pubsub.capnp".Router,
        root :import "anchor.capnp".Anchor,
        # TODO(soon) ...
    );
}


interface Signer {
    sign @0 (challenge :Data) -> (signed :Data);
}
