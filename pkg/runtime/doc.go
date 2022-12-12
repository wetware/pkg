/*
Package runtime provides a high-level API for constructing Wetware
clients and servers using dependency injection.

The runtime package is intended to be used with go.uber.org/fx.  As
such, it provides constructors for the fx.Option type, which can be
consumed by fx.New to create a client or server node.  Refer to the
Fx documentation for information about how to consume fx.Options.

The runtime package exports three basic types:

 1. runtime.Env is a configuration struct containing top-level types
    required by the runtime.  These types are effectful, interacting
    with the host environment:  loggers, metrics, contexts, configs,
    etc.  The Options() method exports these types to Fx.

 2. runtime.Config contains options for the libp2p, CASM and Wetware
    constructors required to build a node. These are passed to their
    constructors lazily, via the fx.Options returned by the Client()
    and Server() methods. The Options returned by either Client() or
    Server() MUST be passed to fx.New() along with those returned by
    runtime.Env.Options().

 3. runtime.Option allows callers to pass options to libp2p, CASM or
    Wetware. These are initially staged in runtime.Config, whereupon
    they are passed into the fx.Options produced by the Client() and
    Server() methods.

The runtime API also exports two high-level constructors for building
client and server nodes: NewClient() and NewServer().  Callers SHOULD
prefer these over manual invocation of Env.Options(), Config.Client()
and Config().Server(), as they apply sensible defaults and are easier
to use overall.  The lower-level API is provided for advanced users.
*/
package runtime
