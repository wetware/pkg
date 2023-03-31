using Go = import "/go.capnp";

@0xfcba4f486a351ac3;

$Go.package("service");
$Go.import("github.com/wetware/ww/internal/api/service");


struct Record {
    serviceName   @0 :Text;
    # ServiceName is used as the signature domain for the record.
    # This means that record created for one service cannot be reused
    # for another, preventing cross-domain forgery attacks.
    
    union{
        sturdyRef @1 :import "cluster.capnp".SturdyRef;
        # SturdyRef is a persistent pointer to a capability a located
        # in a specific vat. It can be passed to cluster.Host.resolve
        # to obtain the corresponding capability.

        multiaddr @2 :Data;
        # Multiaddr contains a binary-encoded multiaddr. This provides
        # a general-purpose format for exporting services that are not
        # implemented using Wetware primitives, such as an HTTP server.
        #
        # For services that export a capnp capability from a Wetware-
        # compatible vat, use of SturdyRef is RECOMMENDED.

        anchor   @3 :Text;
        # Anchor path is a (possibly relative) path to an anchor.

        customStruct   @4 :AnyStruct;
        # Arbitrary, application-defined data.  The struct MUST NOT
        # contain any capabilities.

        customList @5 :AnyList;
        # Arbitrary, application-defined data.  The list MUST NOT
        # contain any capabilities.

        customText @6 :Text;
        customData @7 :Data;
    }
}

