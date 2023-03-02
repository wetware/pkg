using Go = import "/go.capnp";

@0x872a451f9aa74ebf;

$Go.package("channel");
$Go.import("github.com/wetware/ww/internal/api/channel");


interface Closer {
    close @0 ();
}

interface Sender(T) {
    send  @0 (value :T) -> ();
}

interface Recver(T) {
    recv  @0 () -> (value :T);
}

interface SendCloser(T) extends(Sender(T), Closer) {
    asSender @0 () -> (sender :Sender(T));
    asCloser @1 () -> (closer :Closer);
}

interface Chan(T) extends(SendCloser(T), Recver(T)) {
    asSendCloser @0 () -> (sendCloser :SendCloser(T));
    asRecver     @1 () -> (recver :Recver(T));
}
