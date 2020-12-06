// Package shell contains the `ww shell` command implementation.
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/chzyer/readline"
	"github.com/spy16/slurp/repl"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	clientutil "github.com/wetware/ww/internal/util/client"
	ctxutil "github.com/wetware/ww/internal/util/ctx"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang"
	"github.com/wetware/ww/pkg/lang/core"
	"github.com/wetware/ww/pkg/lang/reader"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

const bannerTemplate = `Wetware v{{.App.Version}}
Copyright {{.App.Copyright}}
Compiled with {{.GoVersion}} for {{.GOOS}}
`

var (
	flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "join",
			Aliases: []string{"j"},
			Usage:   "connect to cluster through specified peers",
			EnvVars: []string{"WW_JOIN"},
		},
		&cli.StringFlag{
			Name:    "discover",
			Aliases: []string{"d"},
			Usage:   "automatic peer discovery settings",
			Value:   "/mdns",
			EnvVars: []string{"WW_DISCOVER"},
		},
		&cli.StringFlag{
			Name:    "namespace",
			Aliases: []string{"ns"},
			Usage:   "cluster namespace (must match dial host)",
			Value:   "ww",
			EnvVars: []string{"WW_NAMESPACE"},
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "suppress banner message on interactive startup",
			EnvVars: []string{"WW_QUIET"},
		},
		&cli.BoolFlag{
			Name:    "dial",
			Usage:   "dial into a cluster using -join and -discover",
			EnvVars: []string{"WW_AUTODIAL"},
		},
		&cli.DurationFlag{
			Name:  "timeout",
			Usage: "timeout for -dial",
			Value: time.Second * 10,
		},
		&cli.StringSliceFlag{
			Name:    "path",
			Usage:   "location of ww source files",
			Value:   cli.NewStringSlice("~/.ww"),
			EnvVars: []string{"WW_PATH"},
		},

		// debug flags (hidden)
		&cli.BoolFlag{
			Name:   "log-fx",
			Usage:  "output fx dependency injection logs",
			Hidden: true,
		},
	}
)

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "shell",
		Usage:  "start an interactive REPL session",
		Flags:  flags,
		Action: run(),
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		app := fx.New(fxLogger(c),
			fx.Supply(c,
				prompt{Standard: "ww »", Multiline: "   ›"}),
			fx.Provide(
				newPaths,
				newInput,
				newBanner,
				newWriter,
				newPrinter,
				newEvaluator,
				newRootAnchor,
				newReaderFactory),
			fx.Invoke(loop))

		if err := app.Start(ctx); err != nil {
			return err
		}

		return app.Stop(ctx)
	}
}

func loop(f replFactory) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return f.New().Loop(ctx)
}

type replFactory struct {
	fx.In

	Eval      repl.Evaluator
	Banner    string `name:"banner"`
	Prompt    string `name:"prompt"`
	Multiline string `name:"multiline"`
	Stdout    io.Writer
	NewReader repl.ReaderFactory
	Input     repl.Input
	Printer   repl.Printer
}

func (f replFactory) New() *repl.REPL {
	return repl.New(f.Eval,
		repl.WithBanner(f.Banner),
		repl.WithReaderFactory(f.NewReader),
		repl.WithPrompts(f.Prompt, f.Multiline),
		repl.WithInput(f.Input, nil),
		repl.WithOutput(f.Stdout),
		repl.WithPrinter(f.Printer),
	)
}

func newPaths(c *cli.Context) (ps []string, err error) {
	var usr *user.User
	for _, p := range c.StringSlice("path") {
		if p[0] == '~' {
			if usr, err = user.Current(); err != nil {
				break
			}

			p = strings.Replace(p, "~", usr.HomeDir, 1)
		}

		ps = append(ps, filepath.Clean(p))
	}

	return
}

func newRootAnchor(c *cli.Context, lx fx.Lifecycle) (ww.Anchor, error) {
	if !c.Bool("dial") {
		return nopAnchor{}, nil

	}

	ctx := ctxutil.WithDefaultSignals(context.Background())
	ctx, cancel := context.WithTimeout(ctx, c.Duration("timeout"))
	defer cancel()

	root, err := clientutil.Dial(ctx, c)
	if err == nil {
		lx.Append(closehook(root))
	}

	return root, err
}

func newReaderFactory() repl.ReaderFactory {
	return repl.ReaderFactoryFunc(reader.New)
}

func newWriter(c *cli.Context) io.Writer { return c.App.Writer }

func newPrinter() repl.Printer { return printer{} }

func newEvaluator(root ww.Anchor, paths []string) (repl.Evaluator, error) {
	return lang.New(root, paths...)
}

func newInput(c *cli.Context, lx fx.Lifecycle) (repl.Input, error) {
	r, err := readline.NewEx(&readline.Config{
		HistoryFile: "/tmp/ww.tmp", // TODO(enhancement): ~/.ww/history.ww
		Stdout:      c.App.Writer,
		Stderr:      c.App.ErrWriter,

		InterruptPrompt: "⏎",
		EOFPrompt:       "(exit)",

		/*
			TODO(enhancemenbt):  pass in the lang.Ww and configure autocomplete.
								 The lang.Ww instance will need to supply completions.
		*/
		// AutoComplete: completer(ww),
	})

	if err == nil {
		lx.Append(closehook(r))
	}

	return linereader{r}, err
}

type linereader struct{ *readline.Instance }

func (l linereader) Readline() (line string, err error) {
	for {
		if line, err = l.Instance.Readline(); err == readline.ErrInterrupt {
			return "", nil
		}

		return
	}
}

type prompt struct {
	fx.Out

	Standard  string `name:"prompt"`
	Multiline string `name:"multiline"`
}

type printer struct{}

func (printer) Fprintln(w io.Writer, val interface{}) error {
	if err, ok := val.(error); ok {
		_, err = fmt.Fprintf(w, "%+v\n", err)
		return err
	}

	s, err := core.Render(val.(ww.Any))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, s)
	return err
}

type banner struct {
	fx.Out

	Banner string `name:"banner"`
}

func newBanner(c *cli.Context) (b banner, err error) {
	if c.Bool("quiet") {
		return
	}

	var buf bytes.Buffer
	templ := template.Must(template.New("banner").Parse(bannerTemplate))
	if err = templ.Execute(&buf, struct {
		*cli.Context
		GoVersion, GOOS string
	}{
		Context:   c,
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
	}); err == nil {
		b.Banner = buf.String()
	}

	return
}

type nopAnchor []string

func (a nopAnchor) Name() string {
	if anchorpath.Root(a) {
		return ""
	}

	return a[len(a)-1]
}

func (a nopAnchor) Path() []string { return a }

func (nopAnchor) Ls(context.Context) ([]ww.Anchor, error) {
	return []ww.Anchor{}, nil
}

func (a nopAnchor) Walk(_ context.Context, path []string) ww.Anchor {
	return append(a, path...)
}

func (a nopAnchor) Load(context.Context) (ww.Any, error) {
	// TODO:  return something for /

	return nil, errors.New("not found")
}

func (a nopAnchor) Store(context.Context, ww.Any) error {
	if anchorpath.Root(a) {
		return errors.New("not implemented")
	}

	return errors.New("not found")
}

func (a nopAnchor) Go(context.Context, ...ww.Any) (ww.Any, error) {
	return nil, errors.New("not implemented")
}

func fxLogger(c *cli.Context) fx.Option {
	if c.Bool("log-fx") {
		return fx.Options()
	}

	return fx.NopLogger
}

func closehook(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
