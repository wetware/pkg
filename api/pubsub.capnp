using Go = import "/go.capnp";

@0xf9d8a0180405d9ed;

$Go.package("pubsub");
$Go.import("github.com/wetware/pkg/api/pubsub");


interface Topic {
    publish   @0 (msg :Data) -> stream;
    subscribe @1 (consumer :Consumer, buf :UInt16 = 32) -> ();
    name      @2 () -> (name :Text);

    interface Consumer {
        consume @0 (msg :Data) -> stream;
    }
}


interface Router {
    join @0 (name :Text) -> (topic :Topic);
}
