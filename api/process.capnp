using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/pkg/api/process");

using Cid = Data;
using Pid = UInt32;

interface BytecodeCache {
    # BytecodeCache is used to store WASM byte code. May be implemented with
    # anchors or any other means.
    put @0 (bytecode :Data) -> (cid :Data);
    # Put stores the bytecode and returns the cid of the submitted bytecode.
    get @1 (cid :Cid) -> (bytecode :Data);
    # Get returns the bytecode matching a cid if there's a match, null otherwise.
    has @2 (cid :Cid) -> (has :Bool);
    # Has returns true if a bytecode identified by the cid has been previously stored.
}

interface Process {
    # Process is a points to a running WASM process.
    wait   @0 () -> (exitCode :UInt32);
    # Wait until a process finishes running.
    kill   @1 () -> ();
    # Kill the process.
    link   @2 (other :Process, roundtrip :Bool) -> ();
    # Link a process.
    unlink @3 (other :Process, roundtrip :Bool) -> ();
    # Unlink a process.
    linkLocal   @4 (other :Pid) -> ();
    # Link a local process.
    unlinkLocal @5 (other :Pid) -> ();
    # Unlink a local process.
    monitor @6 () -> (event :Text);
    # Monitor a process.
    pause  @7 () -> ();
    # Pause a process.
    resume @8 () -> ();
    # Resume a paused process.
    id @9 () -> (id :Int64);
}

struct Info {
    pid  @0 :Pid;
    ppid @1 :Pid;
    cid  @2 :Cid;
    argv @3 :List(Text);
    time @4 :Int64;
}

interface Events {
    # Events are sent to the WASM process. It's the process' responsiblity to
    # handle events.
    pause  @0 () -> ();
    resume @1 () -> ();
}
