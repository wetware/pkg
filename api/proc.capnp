using Go = import "/go.capnp";

@0xd78885a0de56b292;

$Go.package("proc");
$Go.import("github.com/wetware/ww/internal/api/proc");

using Chan = import "channel.capnp";


interface Executor(T) {
    exec @0 (param :T) -> (proc :Waiter);
}

# Waiter is the basic interface to an asynchronous process.
# It allows callers to block until the process has terminated.
interface Waiter {
    wait @0 () -> ();
}


interface Unix extends(Executor(Command)) {
    struct Command {
        path @0 :Text;
        dir  @1 :Text;
        args @2 :List(Text);
        env  @3 :List(Text);

        stdin  @4 :StreamReader;
        stdout @5 :StreamWriter;
        stderr @6 :StreamWriter;
    }

    interface Proc extends(Waiter) {
        signal @0 (signal :Signal);
        enum Signal {
            sigINT  @0;
            sigTERM @1;
            sigKILL @2;
        }
    }

    interface StreamReader {
        setDst @0 (dst :StreamWriter);
    }

    interface StreamWriter extends(Chan.SendCloser(Data)) {}
}
