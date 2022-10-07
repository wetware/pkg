using Go = import "/go.capnp";

@0xd78885a0de56b292;

$Go.package("proc");
$Go.import("github.com/wetware/ww/internal/api/proc");

using Chan = import "channel.capnp";


interface Executor(Config, Result) {
    exec @0 (config :Config) -> (proc :Waiter(Result));
}


# Waiter is the basic interface to an asynchronous process.
# It allows callers to block until the process has terminated.
interface Waiter(Result) {
    wait @0 () -> (result :Result);
}


## Unix executor that spawns OS processes through a POSIX-like API.
#interface Unix extends(Executor(Command, Proc)) {
#    struct Command {
#        path @0 :Text;
#        dir  @1 :Text;
#        args @2 :List(Text);
#        env  @3 :List(Text);
#
#        stdin  @4 :IOStream.Provider;
#        stdout @5 :IOStream.Stream;
#        stderr @6 :IOStream.Stream;
#
#        using IOStream = import "iostream.capnp";
#    }
#
#    interface Proc extends(Waiter) {
#        signal @0 (signal :Signal);
#        enum Signal {
#            sigINT  @0;
#            sigTERM @1;
#            sigKILL @2;
#        }
#    }
#}
