package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/metrics"
	logutil "github.com/wetware/ww/internal/util/log"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	"github.com/wetware/ww/pkg/server"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

/****************************************************************
 *                                                              *
 *  runtime.go defines core types for the application runtime.  *
 *                                                              *
 ****************************************************************/

// NewClient returns Fx options for a client runtime.  Options
// are passed directly to Client().
func NewClient(ctx context.Context, fs Flags, opt ...Option) fx.Option {
	return fx.Options(
		Env{Ctx: ctx, Flag: fs}.Options(),
		Config{}.With(clientDefaults(opt)).Client(),
	)
}

// NewClient returns Fx options for a server runtime.  Options
// are passed directly to Server().
func NewServer(ctx context.Context, fs Flags, opt ...Option) fx.Option {
	return fx.Options(
		Env{Ctx: ctx, Flag: fs}.Options(),
		Config{}.With(serverDefaults(opt)).Server(),
	)
}

func clientDefaults(opt []Option) []Option {
	return append([]Option{
		WithHostConfig(casm.Client),
	}, opt...)
}

func serverDefaults(opt []Option) []Option {
	return append([]Option{
		WithHostConfig(casm.Server),
	}, opt...)
}

// Flags are used to query configuration parameters.
type Flags interface {
	Bool(string) bool
	IsSet(string) bool
	Path(string) string
	String(string) string
	StringSlice(string) []string
	Duration(string) time.Duration
	Float64(string) float64
}

// Env is a context object that exposes environmental data and
// effectful operations to the runtime.
type Env struct {
	fx.In

	Ctx     context.Context `optional:"true"`
	Log     log.Logger      `optional:"true"`
	Metrics metrics.Client  `optional:"true"`
	Flag    Flags
}

// Options for the Wetware runtime.  These MUST be passed to
// the top-level call to fx.New, along with either options
// provided by either Config.Client() or Config.Server().
func (env Env) Options() fx.Option {
	return fx.Options(
		fx.WithLogger(newFxLogger),
		fx.Supply(
			fx.Annotate(env.Flag, fx.As(new(Flags))),
			fx.Annotate(env.context(), fx.As(new(context.Context))),
			fx.Annotate(env.logging(), fx.As(new(log.Logger))),
			fx.Annotate(env.metrics(), fx.As(new(metrics.Client)))),
		fx.Decorate(
			decorateLogger))
}

func (env Env) context() context.Context {
	if env.Ctx == nil {
		env.Ctx = context.Background()
	}

	return env.Ctx
}

func (env *Env) logging() log.Logger {
	if env.Log == nil {
		env.Log = logutil.New(env.Flag)
	}

	return env.Log
}

func (env *Env) metrics() metrics.Client {
	if env.Metrics == nil {
		env.Metrics = statsdutil.New(env.Flag, env.logging())
	}

	return env.Metrics
}

/*
	Fx Logger
*/

type fxLogger struct{ log.Logger }

func newFxLogger(env Env) fxevent.Logger {
	return fxLogger{env.Log}
}

func (lx fxLogger) LogEvent(ev fxevent.Event) {
	switch event := ev.(type) {
	case *fxevent.LoggerInitialized:
		lx.MaybeError(event.Err).Trace("initialized logger")

	case *fxevent.Supplied:
		lx.MaybeError(event.Err).
			MaybeModule(event.ModuleName).
			WithField("type", event.TypeName).
			Tracef("supplied value of type %s", event.TypeName)

	case *fxevent.Provided:
		lx.Logger = lx.
			MaybeModule(event.ModuleName).
			WithField("fn", event.ConstructorName)

		for _, name := range event.OutputTypeNames {
			lx.WithField("provides", name).
				Tracef("provided constructor for %s", name)
		}

		if event.Err != nil {
			lx.WithError(event.Err).
				Error("error encountered while providing constructor")
		}

	case *fxevent.Replaced:
		lx.Logger = lx.MaybeModule(event.ModuleName)

		for _, name := range event.OutputTypeNames {
			lx.WithField("replaces", name).
				Tracef("replaced %s", name)
		}

		if event.Err != nil {
			lx.WithError(event.Err).
				Error("error encountered while replacing value")
		}

	case *fxevent.Decorated:
		lx.Logger = lx.MaybeModule(event.ModuleName)

		for _, name := range event.OutputTypeNames {
			lx.WithField("decorates", name).
				Tracef("decorated %s", name)
		}

		if event.Err != nil {
			lx.WithError(event.Err).
				Error("error encountered while decorating type")
		}

	case *fxevent.Invoking:
		lx.MaybeModule(event.ModuleName).
			WithField("fn", event.FunctionName).
			Trace("invoking function")

	case *fxevent.Invoked:
		if event.Err != nil {
			lx.MaybeModule(event.ModuleName).
				WithError(event.Err).
				WithField("fn", event.FunctionName).
				Error("function invocation failed")
			fmt.Fprintln(os.Stderr, event.Trace)
		}

	case *fxevent.RollingBack:
		lx.WithError(event.StartErr).
			Error("rolling back")

	case *fxevent.RolledBack:
		if event.Err == nil {
			lx.Debug("rolled back successfully")
			return
		}

		lx.WithError(event.Err).
			Error("rollback failed")

	case *fxevent.Started:
		if event.Err == nil {
			lx.Trace("application started")
			return
		}

		lx.WithError(event.Err).
			Error("application failed to start")

	case *fxevent.Stopping:
		lx.WithField("signal", event.Signal).
			Trace("signal received")

	case *fxevent.Stopped:
		lx.MaybeError(event.Err).
			Trace("runtime stopped")

	case *fxevent.OnStartExecuting:
		lx.WithField("fn", event.FunctionName).
			WithField("caller", event.CallerName).
			Trace("executing start hook")

	case *fxevent.OnStartExecuted:
		lx.Logger = lx.MaybeError(event.Err).
			WithField("fn", event.FunctionName).
			WithField("caller", event.CallerName).
			WithField("dur", event.Runtime)

		if event.Err == nil {
			lx.Trace("exeuted start hook")
		} else {
			lx.Error("start hook failed")
		}

	case *fxevent.OnStopExecuting:
		lx.WithField("fn", event.FunctionName).
			WithField("caller", event.CallerName).
			Trace("executing stop hook")

	case *fxevent.OnStopExecuted:
		lx.Logger = lx.MaybeError(event.Err).
			WithField("fn", event.FunctionName).
			WithField("caller", event.CallerName).
			WithField("dur", event.Runtime)

		if event.Err == nil {
			lx.Trace("exeuted start hook")
		} else {
			lx.Error("start hook failed")
		}

	default:
		panic(fmt.Sprintf("invalid Fx event: %s", reflect.TypeOf(ev)))
	}
}

func (lx fxLogger) MaybeError(err error) fxLogger {
	if err != nil {
		lx.Logger = lx.WithError(err)
	}

	return lx
}

func (lx fxLogger) MaybeModule(name string) fxLogger {
	if name != "" {
		lx.Logger = lx.WithField("module", name)
	}

	return lx
}

/*
	Misc.
*/

func decorateLogger(env Env, vat casm.Vat) log.Logger {
	log := env.Log.With(vat)

	if env.Flag.IsSet("meta") {
		log = log.WithField("meta", env.Flag.StringSlice("meta"))
	}

	return log
}

func newServerNode(j server.Joiner, ps *pubsub.PubSub) (*server.Node, error) {
	// TODO:  this should use lx.OnStart to benefit from the start timeout.
	return j.Join(ps)
}

func bootServer(lx fx.Lifecycle, n *server.Node) {
	lx.Append(fx.Hook{
		OnStart: bootstrap(n),
		OnStop:  onclose(n),
	})
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: onclose(c),
	}
}

func onclose(c io.Closer) func(context.Context) error {
	return func(context.Context) error {
		return c.Close()
	}
}

type bootstrappable interface {
	Bootstrap(context.Context) error
}

func bootstrap(b bootstrappable) func(context.Context) error {
	return func(ctx context.Context) error {
		return b.Bootstrap(ctx)
	}
}
