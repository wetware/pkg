using Go = import "/go.capnp";

@0xfa005a3c690f4a62;

$Go.package("boot");
$Go.import("github.com/wetware/pkg/api/boot");


struct Packet {
    namespace         @0 :Text;

    union {
        request          :group {
            from      @1 :PeerID;
        }

        survey           :group {
            from      @2 :PeerID;
            distance  @3 :UInt8;
        }

        response         :group {
            peer      @4 :PeerID;
            addrs     @5 :List(Multiaddr);
        }
    }

    using PeerID   = Text;
    using Multiaddr = Data;
}
