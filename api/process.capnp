using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/ww/internal/api/process");


interface Executor {
    spawn @0 (byteCode :Data, entryPoint :Text) -> (process :Process);
    # spawn a WASM based process from the binary module with the target
    # entry function 

    using IOStream = import "iostream.capnp";
}

interface Process {
    start  @0 () -> ();             # start the process
    stop   @1 () -> ();              # TODO: provide a signal such as SIGTERM, SIGKILL...
    wait   @2 () -> (error :Text);   # wait for an started process to finish
    
    stdin  @3 () -> (stdin :IOStream.Stream);
    stdout @4 (stdout :IOStream.Stream) ->  ();
    stderr @5 (stderr :IOStream.Stream) -> ();
    using IOStream = import "iostream.capnp";
}
