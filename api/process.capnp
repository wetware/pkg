using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/ww/internal/api/process");


interface Executor {
    spawn @0 (byteCode :Data, entryPoint :Text = "run") -> (process :Process);
    # spawn a WASM based process from the binary module with the target
    # entry function 
}

interface Process {
    start  @0 () -> ();
    stop   @1 () -> ();
    wait   @2 () -> (exitCode :UInt32);
}
