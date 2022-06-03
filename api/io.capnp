using Go = import "/go.capnp";

@0xeba464918d53d496;

$Go.package("io");
$Go.import("github.com/wetware/ww/internal/api/io");


interface Reader {
    read @0 (n :UInt16) -> (data :Data, err :Error);
}

interface Writer {
    write @0 (data :Data) -> (n :Int64, err :Error);
}

interface Closer {
    close @0 () -> ();
}

interface ReadCloser      extends(Reader, Closer)         {}
interface WriteCloser     extends(Writer, Closer)         {}
interface ReadWriter      extends(Reader, Writer)         {}
interface ReadWriteCloser extends(Reader, Writer, Closer) {}


struct Error {
    code    @0 :Code;
    message @1 :Text;
    
    enum Code {
        # Where possible, error codes are used in lieu of
        # text.   This allows generated code to translate
        # common errors into a native language exception,
        # or other error type. In Go, these are mapped to
        # their respective errors from the "io" package.
        #
        # Note that the 'nil' case does not always signal
        # success.   Errors that do not map directly onto
        # these codes are reported using a message string
        # with code set to 'nil'.  A call is successful iff
        # the code is set to nil *and* the message is empty.
        nil              @0;
        shortWrite       @1;
        shortBuf         @2;
        eof              @3;
        unexpectedEOF    @4;
        noProgress       @5;

        # We also include network errors ...
        closed           @6;

        # ... along with context errors ...
        canceled         @7;
        deadlineExceeded @8;  # also matches os.ErrDeadlineExceeded
    }
}