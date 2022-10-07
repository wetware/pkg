using Go = import "/go.capnp";

@0x9419a7a54f76d35b;

$Go.package("wasm");
$Go.import("github.com/wetware/ww/internal/api/wasm");


interface Runtime extends(Executor(Config, Proc)) {
    struct Config {
        src      @0 :Data;
        
        stdin    @1 :IOStream.Provider;
        stdout   @2 :IOStream.Stream;
        stderr   @3 :IOStream.Stream;
        
        randSeed @4 :UInt64;

        using IOStream = import "iostream.capnp";
    }

    interface Proc {
        close @0 (exitCode :UInt32) -> ();
        # Close all the modules that have been initialized in this Runtime
        # with the provided exit code.  An error is returned if any module
        # returns an error when closed.
    }
    
    using Executor = import "proc.capnp".Executor;
}
