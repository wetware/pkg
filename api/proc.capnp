using Go = import "/go.capnp";

@0xd78885a0de56b292;

$Go.package("proc");
$Go.import("github.com/wetware/ww/internal/api/proc");

using Chan = import "channel.capnp";


interface Executor(T) {
    exec @0 (param :T) -> (proc :P);
}

# P is a basic asynchronous process capability.  
interface P {
    wait @0 () -> ();
}


interface Unix extends(Executor(Command)) {
    struct Command {
        path @0 :Text;
        dir  @1 :Text;
        args @2 :List(Text);
        env  @3 :List(Text);

        stdin  @4 :InputStream;
        stdout @5 :OutputStream;
        stderr @6 :OutputStream;
    }

    interface InputStream  extends(Chan.Recver(Data)) {}
    interface OutputStream extends(Chan.SendCloser(Data)) {}
}
