using Go = import "/go.capnp";

@0xd78885a0de56b292;

$Go.package("proc");
$Go.import("github.com/wetware/ww/internal/api/proc");

interface UnixExecutor {
    command @0 (name :Text, arg :List(Text)) -> (cmd :Cmd);
}

interface Cmd {
    start @0 () -> (err :Text);
    wait @1 () -> (err :Text);
    stderrPipe @2 () -> (rc :ReadCloser);
    stdinPipe @3 () -> (wc :WriteCloser);
    stdoutPipe @4 () -> (rc :ReadCloser);
}

interface ReadCloser extends(Reader, Closer){}

interface WriteCloser extends(Writer, Closer){}

interface Reader {
    read @0 (n: Int64) -> (p :Data, n: Int64);
}

interface Writer {
    write @0 (p :Data) -> (n :Int64);
}

interface Closer {
    close @0 () -> ();
}
