using Go = import "/go.capnp";

@0x872a451f9aa74ebf;

$Go.package("channel");
$Go.import("github.com/wetware/ww/internal/api/channel");


interface Closer {
    close @0 () -> ();
}


interface Sender(T) {
    send  @0 (value :T) -> ();
}


interface Peeker(T) {
    peek @0 () -> (value :T);
}


interface Recver(T) {
    recv  @0 () -> (value :T);
}


interface SendCloser(T) extends(Sender(T), Closer) {}
interface PeekRecver(T) extends(Peeker(T), Recver(T)) {}
interface Chan(T) extends(SendCloser(T), Recver(T)) {}
interface PeekableChan(T) extends(SendCloser(T), PeekRecver(T)) {}
