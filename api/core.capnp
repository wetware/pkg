using Go = import "/go.capnp";

@0xe82706a772b0927b;

$Go.package("core");
$Go.import("github.com/wetware/pkg/api/core");

using CapStore = import "capstore.capnp";
using Cluster = import "cluster.capnp";
using Process = import "process.capnp";


interface Terminal {
    login @0 (account :Cluster.Signer) -> (session :Session);
}

# Session is a capability-set that was granted to a particular
# user.  It is the application wide ambient-authority boundary.
struct Session {
    local         :group{
        peer   @0 :Text;    # peer.ID
        server @1 :UInt64;  # routing.ID
        host   @2 :Text;    # hostname
    }

    # Access-controlled capabilities.  These will be set to null
    # unless permission has been granted to use the object.
    view       @3 :Cluster.View;
    exec       @4 :Executor;
    capStore   @5 :CapStore.CapStore;
    extra      @6 :List(Extra);

    struct Extra {
        name   @0 :Text;
        client @1 :Capability;
    }
}

interface Executor {
    # Executor has the ability to create and run WASM processes given the
    # WASM bytecode.
    exec @0 (bytecode :Data, ppid :UInt32, bctx :Process.BootContext) -> (process :Process.Process);
    # Exec creates an runs a process from the provided bytecode. Optionally, a
    # capability can be passed through the `cap` parameter. This capability will
    # be available at the process bootContext.
    #
    # The Process capability is associated to the created process.
    execCached @1 (cid :Data, ppid :UInt32, bctx :Process.BootContext) -> (process :Process.Process);
    # Same as Exec, but the bytecode is directly from the BytecodeRegistry.
    # Provides a significant performance improvement for medium to large
    # WASM streams.
}

