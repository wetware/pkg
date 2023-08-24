//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:anchor anchor.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:bitswap bitswap.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:boot boot.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:capstore capstore.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:channel channel.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:cluster cluster.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:process process.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:pubsub pubsub.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/std -ogo:registry registry.capnp

package api
