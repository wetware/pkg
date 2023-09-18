using Go = import "/go.capnp";

@0xe82706a772b0927b;

$Go.package("core");
$Go.import("github.com/wetware/pkg/api/core");

using CapStore = import "capstore.capnp";
using Process = import "process.capnp";


# Signer identifies an accound.  It is a capability that can be
# used to sign arbitrary nonces.
#
# The signature domain is "ww.auth"
interface Signer {
    sign @0 (challenge :Data) -> (signed :Data);
}


interface Terminal {
    login @0 (account :Signer) -> (session :Session);
}

# Session is a capability-set that was granted to a particular
# user.  It is the application wide ambient-authority boundary.
struct Session {
    view        @0 :import "cluster.capnp".View;
    local          :group{
        peer    @1 :Text;    # peer.ID
        server  @2 :UInt64;  # routing.ID
        host    @3 :Text;    # hostname
    }
}


interface Executor {
    # Executor has the ability to create and run WASM processes
    # given the WASM bytecode.
    exec @0 (session :Session, bytecode :Data, ppid :UInt32, args :List(Text)) -> (process :Process.Process);
    # Exec creates an runs a process from the provided bytecode.
    #
    # The Process capability is associated to the created process.
    execCached @1 (session :Session, cid :Data, ppid :UInt32, args :List(Text)) -> (process :Process.Process);
    # Same as Exec, but the bytecode is directly from the BytecodeRegistry.
    # Provides a significant performance improvement for medium to large
    # WASM streams.
}

