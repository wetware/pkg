using Go = import "/go.capnp";

@0xf9d8a0180405d9ed;

$Go.package("pubsub");
$Go.import("github.com/wetware/ww/internal/api/pubsub");


interface Topic {
    publish   @0 (msg :Data) -> ();
    subscribe @1 (handler :Handler) -> ();

    interface Handler {
        handle @0 (msg :Data) -> ();
    }
}


interface PubSub {
    join @0 (name :Text) -> (topic :Topic);
}


# interface PubSub {
#     join @0 (name :Text) -> (topic :Topic);
#     interface Topic {
#         relay @0 () -> ();
#     }
# 
#     interface Subscriber extends(Topic) {
#         subscribe @0 (recvr :Recver) -> ();
#         interface Recver {
#             recv @0 (event :Event) -> ();
#             struct Event {
#                 union {
#                     message @0 :Data;
#                     closed  @1 :Void;
#                 }
#             }
#         }
#     }
# 
#     interface Publisher extends(Topic) {
#         publish @0 (msg :Data) -> ();
#     }
# }
