package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	"github.com/lthibault/log"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/metrics"
	logutil "github.com/wetware/ww/internal/util/log"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
)

/****************************************************************
 *                                                              *
 *  runtime.go defines core types for the application runtime.  *
 *                                                              *
 ****************************************************************/

// Env is a context object that exposes environmental data and
// effectful operations to the runtime.
type Env interface {
	// Context returns the main runtime context.  The returned
	// context MAY expire, in which case callers SHOULD finish
	// any outstanding work and terminate promptly.
	Context() context.Context

	/*
		Observability
	*/

	Log() log.Logger
	Metrics() metrics.Client

	/*
		Configuration
	*/

	Flags
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

func NewEnv(ctx context.Context, fs Flags) Env {
	logging := logutil.New(fs)
	metrics := statsdutil.New(fs, logging)

	return &basicEnv{
		Flags:   fs,
		context: ctx,
		logging: logging,
		metrics: metrics,
	}
}

type basicEnv struct {
	Flags
	context context.Context
	logging log.Logger
	metrics metrics.Client
}

func (env basicEnv) Context() context.Context {
	return env.context
}

func (env basicEnv) Log() log.Logger {
	return env.logging
}

func (env basicEnv) Metrics() metrics.Client {
	return env.metrics
}

// Prelude provides the core wetware runtime.  It MUST be passed
// to the top-level call to fx.New.
func Prelude(env Env) fx.Option {
	return fx.Options(fx.WithLogger(newFxLogger),
		fx.Supply(
			fx.Annotate(env, fx.As(new(Env))),
			fx.Annotate(env.Log(), fx.As(new(log.Logger))),
			fx.Annotate(env.Metrics(), fx.As(new(metrics.Client)))),
		fx.Decorate(
			decorateLogger,
			decorateEnv))
}

type fxLogger struct{ log.Logger }

func newFxLogger(env Env) fxevent.Logger {
	return fxLogger{env.Log()}
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

type environment struct {
	log log.Logger
	Env
}

func decorateEnv(env Env, log log.Logger) Env {
	return environment{
		log: log,
		Env: env,
	}
}

func (env environment) Log() log.Logger {
	return env.log
}

func decorateLogger(env Env, vat casm.Vat) log.Logger {
	log := env.Log().With(vat)

	if env.IsSet("meta") {
		log = log.WithField("meta", env.StringSlice("meta"))
	}

	return log
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
