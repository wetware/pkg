using Go = import "/go.capnp";

@0x81484d9336a7c5d3;

$Go.package("auth");
$Go.import("github.com/wetware/pkg/api/auth");


struct Session {
    view   @0 :import "cluster.capnp".View;
    root   @1 :import "anchor.capnp".Anchor;
    pubsub @2 :import "pubsub.capnp".Router;
}


interface Terminal {
    login @0 (account :Signer) -> (status :Status);

    struct Status {
        union {
            success @0 :Session;
            failure @1 :Text;
        }
    }
}


interface Signer {
    sign @0 (challenge :Data) -> (signed :Data);
}
