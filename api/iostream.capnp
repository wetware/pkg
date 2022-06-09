using Go = import "/go.capnp";

@0x89c985e63e991441;

$Go.package("iostream");
$Go.import("github.com/wetware/ww/internal/api/iostream");

using Chan = import "channel.capnp";


# Stream can send bytes to a remote vat.
interface Stream extends(Chan.SendCloser(Data)) {}

# Provider can make a Stream available to a remote vat.
interface Provider {
    provide @0 (stream :Stream);
}
