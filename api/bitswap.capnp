using Go = import "/go.capnp";

@0xced7a3b0e18b5291;

$Go.package("bitswap");
$Go.import("github.com/wetware/ww/internal/api/bitswap");


interface BitSwap {
    # BitSwap is a protocol for exchanging data between peers in a
    # P2P network.  It can resolve a content ID (CID) to a blob of
    # bytes efficiently.  BitSwap is optimized for the transfer of
    # large chunks of data.

    getBlock @0 (key :CID) -> (block :Data);
    # GetBlock attempts to retrieve a particular block from peers.
    # Implementations SHOULD provide a user-controlled mechanism to
    # time-out.

    # getBlocks @1 (keys :List(CID), chan :Sender(Data)) -> ();
    # # GetBlocks resolves multiple CIDs, streaming the corresponding
    # # blocks through the provided channel.   Note that there are no
    # # ordering guarantees for blocks.

    # interface Session {
    #     # TODO
    # }

    using CID = Data;
    #using Sender = import "channel.capnp".Sender;
}
