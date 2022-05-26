using Go = import "/go.capnp";

@0xd78885a0de56b292;

$Go.package("proc");
$Go.import("github.com/wetware/ww/internal/api/proc");

interface Executor(T) {
    exec @0 (command :T) -> (proc :Process);
}

interface Process {
    start @0 () -> ();
    wait @1 () -> ();
}

interface ReadCloser extends(Reader, Closer){}

interface WriteCloser extends(Writer, Closer){}

interface Reader {
    read @0 (n: Int64) -> (data :Data, n: Int64);
}

interface Writer {
    write @0 (data :Data) -> (n :Int64);
}

interface Closer {
    close @0 () -> ();
}

interface UnixProcess extends(Process) {
    stderrPipe @0 () -> (rc :ReadCloser);
    stdinPipe @1 () -> (wc :WriteCloser);
    stdoutPipe @2 () -> (rc :ReadCloser);
}

struct UnixCommand {
    name @0 :Text;
    arg @1 :List(Text);
}
