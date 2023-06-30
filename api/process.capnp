using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/ww/api/process");


interface Executor {
    exec @0 (bytecode :Data) -> (process :Process);
    execWithCap @1 (bytecode :Data, cap :Capability) -> (process :Process);  # TODO replace exec with execWithCap
    # exec a WASM based process
}

interface Process {
    wait   @0 () -> (exitCode :UInt32);
    kill   @1 () -> ();
}
