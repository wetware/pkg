using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/ww/api/process");

using Tools = import "/experiments/tools.capnp";


interface Executor {
    # Executor has the ability to create and run WASM processes given the
    # WASM bytecode.
    exec @0 (bytecode :Data, ppid :UInt32,  caps :List(Capability)) -> (process :Process);
    # Exec creates an runs a process from the provided bytecode. Optionally, a
    # capability can be passed through the `cap` parameter. This capability will
    # be available at the process bootContext.
    #
    # The Process capability is associated to the created process.
    execFromCache @1 (hash :Data, ppid :UInt32, caps :List(Capability)) -> (process :Process);
    # Same as Exec, but the bytecode is directly from the BytecodeRegistry.
    # Provides a significant performance improvement for medium to large
    # WASM streams.
    registry @2 () -> (registry :BytecodeRegistry);
    # Registry returns a capability for the BytecodeRegistry of the Executor.
    tools @3 () -> (tools :Tools.Tools);
    # Tools makes the experimental tools accessibles to anyone with access to the
    # Executor capability.
}

interface Args {
    # Args contains a list of strings that can be passed to a process via caps.
    args @0 () -> (args :List(Text));
}

interface Process {
    # Process is a points to a running WASM process.
    wait   @0 () -> (exitCode :UInt32);
    # Wait until a process finishes running.
    kill   @1 () -> ();
    # Kill the process.
}

interface BootContext {
    # BootContext is used to make other capabilities available to spawning processes.
    # e.g. if process A spawns process B, it can leave a channel pointing to A
    # in it's bootContext to stablish a direct communication channel.
    # Every process is given a context when created.
    open @0 () -> (content :List(Capability));
    # Open returns all the capabilities that were left on the bootContext.
    # The receiver must know the order of the content beforehand.
}

interface BytecodeRegistry {
    # BytecodeRegistry is used to store WASM byte code. May be implemented with
    # anchors or any other means.
    put @0 (bytecode :Data) -> (hash :Data);
    # Put stores the bytecode and returns the hash of the submitted bytecode.
    get @1 (hash :Data) -> (bytecode :Data);
    # Get returns the bytecode matching a hash if there's a match, null otherwise.
    has @2 (hash :Data) -> (has :Bool);
    # Has returns true if a bytecode identified by the hash has been previously stored.
}

