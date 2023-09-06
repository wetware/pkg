//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:anchor anchor.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:bitswap bitswap.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:boot boot.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:capstore capstore.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:channel channel.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:cluster cluster.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:process process.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:pubsub pubsub.capnp
//go:generate capnp compile -I$GOPATH/src/capnproto.org/go/capnp/v3/std -ogo:registry registry.capnp

package api
