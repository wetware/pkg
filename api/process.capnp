using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/pkg/api/process");


interface Executor {
    exec @0 (bytecode :Data) -> (process :Process);
    # exec a WASM based process
}

interface Process {
    wait   @0 () -> (exitCode :UInt32);
    kill   @1 () -> ();
}
