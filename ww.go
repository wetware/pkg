package ww

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"runtime"

	// "github.com/spy16/slurp"
	// "github.com/spy16/slurp/core"
	// "github.com/spy16/slurp/reader"
	// "github.com/spy16/slurp/repl"
	"github.com/lthibault/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	casm "github.com/wetware/casm/pkg"

	// "github.com/wetware/ww/api"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/system"
	"go.uber.org/fx"
)

const Version = "0.1.0"

type Ww struct {
	fx.In `ignore-unexported:"true"`

	Log    log.Logger
	Name   string
	Stdin  io.Reader `name:"stdin"`
	Stdout io.Writer `name:"stdout"`
	Stderr io.Writer `name:"stderr"`
	ROM    system.ROM
	Vat    casm.Vat
}

func (ww Ww) String() string {
	return ww.Name
}

// func REPL(ctx context.Context, ww Ww, code ROM) error {
// 	ww.bind(code)

// 	lisp := slurp.New(lang.Analyzer(&ww))
// 	if err := lisp.Bind(lang.Globals(&ww)); err != nil {
// 		return err
// 	}

// 	vm := repl.New(lisp,
// 		repl.WithBanner("Wetware"),
// 		repl.WithReaderFactory(ww.reader(ctx)),
// 		repl.WithInput(ww.input(ctx), ww.mapErr(ctx)),
// 		repl.WithPrinter(ww.printer(ctx)),
// 		repl.WithPrompts("(ww)", "    |"))
// 	return vm.Loop(ctx)
// }

func (ww Ww) Exec(ctx context.Context) error {
	r := wazero.NewRuntimeWithConfig(ctx, wazero.
		NewRuntimeConfigCompiler().
		WithCloseOnContextDone(true))
	c, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	compiled, err := r.CompileModule(ctx, ww.ROM)
	if err != nil {
		return err
	}
	defer compiled.Close(ctx)

	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithStartFunctions(). // don't call _start until later
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithName(ww.Name).
		WithEnv("ns", ww.Name).
		WithStdin(ww.Stdin).
		WithStdout(ww.Stdout).
		WithStderr(ww.Stderr).
		WithFSConfig(wazero.
			NewFSConfig().
			// WithFS(fs).
			WithFSMount(ww, ww.Name))) // mount ww to ./ww
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	if fn := mod.ExportedFunction("_start"); fn != nil {
		_, err = fn.Call(ctx)
		return err
	}

	return errors.New("missing export: _start")
}

// func (ww *Ww) Connect(ctx context.Context, a api.Anchor) (api.Session_Future, capnp.ReleaseFunc) {
// 	f, release := a.Load(ctx, func(load api.Anchor_load_Params) error {
// 		return load.SetAccount(ww.signer())
// 	})

// 	return f.Session(), release // request pipelining FTW!
// }

// func (ww *Ww) signer() api.Signer {
// 	host := ww.Vat.Host
// 	privkey := host.Peerstore().PrivKey(host.ID())
// 	return api.Signer_ServerToClient(&signOnce{
// 		PrivKey: privkey,
// 	})
// }

func (ww Ww) Root() *anchor.Node {
	return &anchor.Node{}
}

func (ww Ww) Open(name string) (fs.File, error) {
	return nil, errors.New("TODO: ww.Open()")

	// path, err := anchor.NewPath(path.Clean(name)).Maybe()
	// if err != nil {
	// 	return nil, err
	// }

	// host, guest := net.Pipe()
	// go func() {
	// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	// 	defer cancel()

	// 	// local, so all good
	// 	f, release := ww.Root().Walk(ctx, func(walk api.Anchor_walk_Params) error {
	// 		return walk.SetPath(path)
	// 	})
	// 	defer release()

	// 	boot, resolver := capnp.NewLocalPromise[api.Anchor]()
	// 	conn := rpc.NewConn(rpc.NewStreamTransport(host), &rpc.Options{
	// 		BootstrapClient: capnp.Client(boot),
	// 		ErrorReporter: errLogger{
	// 			Logger: ww.Log.With(
	// 				slog.String("rpc.Conn", "host")),
	// 		},
	// 	})
	// 	defer conn.Close()

	// 	signer := api.Signer(conn.Bootstrap(ctx))
	// 	ww.authenticate(resolver)

	// 	<-conn.Done() // conn is closed by authenticate if auth fails
	// }()
	// defer ww.Log.Debug("session started")

	// return &lang.Socket{
	// 	NS:   ww.Name,
	// 	Conn: guest,
	// }, nil
}

// func (ww Ww) reader(ctx context.Context) repl.ReaderFactoryFunc {
// 	return func(r io.Reader) *reader.Reader {
// 		rd := reader.New(r)

// 		rd.SetMacro('/', false, lang.PathMacro(ctx))

// 		return rd
// 	}
// }

// func (ww Ww) input(ctx context.Context) repl.Input {
// 	return repl.NewPrompt(repl.NewLineReader(ww.Stdin), ww.Stdout)
// }

// func (ww Ww) mapErr(ctx context.Context) repl.ErrMapper {
// 	logger := ww.Log.WithGroup("ww")

// 	return func(err error) error {
// 		if err != nil {
// 			logger.Error(err.Error())
// 		}

// 		return err // TODO:  effect system goes here?
// 	}
// }

// func (ww Ww) printer(ctx context.Context) repl.Printer {
// 	return &repl.Renderer{
// 		Out: ww.Stdout,
// 		Err: ww.Stderr,
// 	}
// }

// // Print prints val to w.
// func (ww Ww) Print(val interface{}) (err error) {
// 	switch x := val.(type) {
// 	case nil:

// 	case error:
// 		ww.Log.Error(x.Error())

// 	case core.SExpressable:
// 		sexpr, err := x.SExpr()
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Fprintln(ww.Stdout, sexpr)

// 	default:
// 		ww.Log.Info("unhandled value",
// 			slog.Any("value", val),
// 			slog.Any("type", reflect.TypeOf(val)))
// 	}

// 	return
// }

// func (ww Ww) authenticate(signer api.Signer, r capnp.Resolver[api.Anchor]) {
// 	ww.Log.Warn("called stub function 'authenticate()'")

// 	// signer := api.Signer(conn.Bootstrap(ctx))

// }

// // signOnce is an api.Signer that will succeed once, and then return
// // "signer revoked" for all subsequent calls.
// type signOnce struct {
// 	once    sync.Once
// 	PrivKey crypto.PrivKey
// }

// func (s *signOnce) Sign(ctx context.Context, call api.Signer_sign) error {
// 	var r anchor.AuthRecord
// 	binary.BigEndian.PutUint64(r[:], call.Args().Nonce())

// 	res, err := call.AllocResults()
// 	if err != nil {
// 		return err
// 	}

// 	var e *record.Envelope
// 	err = errors.New("signer revoked")
// 	s.once.Do(func() {
// 		if e, err = record.Seal(&r, s.PrivKey); err != nil {
// 			return
// 		}

// 		var b []byte
// 		if b, err = e.Marshal(); err == nil {
// 			res.SetEnvelope(b)
// 		}
// 	})

// 	return err
// }

// type errLogger struct {
// 	*slog.Logger
// }

// func (e errLogger) ReportError(err error) {
// 	if err != nil {
// 		e.Debug("rpc connection failed",
// 			slog.String("error", err.Error()))
// 	}
// }
