using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/pkg/api/process");


interface Executor {
    # Executor has the ability to create and run WASM processes given the
    # WASM bytecode.
    exec @0 (bytecode :Data, bootstrapClient :Capability) -> (process :Process);
    # Exec creates an runs a process from the provided bytecode. Optionally, a
    # capability can be passed through the `cap` parameter. This capability will
    # be available at the process bootContext.
    #
    # The Process capability is associated to the created process.
    execCached @1 (hash :Data, bootstrapClient :Capability) -> (process :Process);
    # Same as Exec, but the bytecode is directly from the BytecodeRegistry.
    # Provides a significant performance improvement for medium to large
    # WASM streams.
}

interface Process {
    # Process is a points to a running WASM process.
    wait   @0 () -> (exitCode :UInt32);
    # Wait until a process finishes running.
    kill   @1 () -> ();
    # Kill the process.
}

interface BytecodeCache {
    # BytecodeCache is used to store WASM byte code. May be implemented with
    # anchors or any other means.
    put @0 (bytecode :Data) -> (hash :Data);
    # Put stores the bytecode and returns the hash of the submitted bytecode.
    get @1 (hash :Data) -> (bytecode :Data);
    # Get returns the bytecode matching a hash if there's a match, null otherwise.
    has @2 (hash :Data) -> (has :Bool);
    # Has returns true if a bytecode identified by the hash has been previously stored.
}
