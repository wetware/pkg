using Go = import "/go.capnp";

@0xf9d8a0180405d9ed;

$Go.package("pubsub");
$Go.import("github.com/wetware/ww/internal/api/pubsub");


interface Topic {
    publish   @0 (msg :Data) -> ();
    subscribe @1 (chan :Sender, opts :SubOpts) -> ();
    name      @2 () -> (name :Text);

    using Sender = import "channel.capnp".Sender;

    struct SubOpts{
        bufferSize @0 :Int64;
    }
}


interface PubSub {
    join @0 (name :Text) -> (topic :Topic);
}
