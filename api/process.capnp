using Go = import "/go.capnp";

@0x9a51e53177277763;

$Go.package("process");
$Go.import("github.com/wetware/ww/internal/api/process");


interface Executor {
    spawn @0 (binary :Data, entryfunction :Text) -> (process :Process);
    # spawn a WASM based process from the binary module with the target
    # entry function 

    using IOStream = import "iostream.capnp";
}

interface Process {
    start @0 () -> ();  # start the process
    stop @1 () -> (); # TODO: provide a signal such as SIGTERM, SIGKILL...
    wait @2 () -> (error :Text);  # wait for an started process to finish
    close @3 () -> ();  # close should always be called after running a process
    
    input @4 () -> (stream :IOStream.Stream);
    # the resulting stream can be used to provide input to the process
    output @5(stream :IOStream.Stream) -> (error :Text);
    # receives an stream to provide output to

    using IOStream = import "iostream.capnp";
}
