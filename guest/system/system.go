package system

import (
	"context"
	"log/slog"
	"os"

	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/system"

	"capnproto.org/go/capnp/v3/rpc"
)

func init() {
	var level = slog.LevelInfo
	switch os.Getenv("WW_DEBUG") {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	root := slog.New(h).With("rom", os.Args[0])
	slog.SetDefault(root)
}

func Login(ctx context.Context) (auth.Session, error) {
	opt := &rpc.Options{
		ErrorReporter: system.ErrorReporter{
			Logger: slog.Default(),
		},
	}

	conn := rpc.NewConn(rpc.NewStreamTransport(socket{}), opt)
	go func() {
		defer conn.Close()
		select {
		case <-conn.Done():
		case <-ctx.Done():
		}
	}()

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return auth.Session{}, err
	}
	term := core.Terminal(client)

	f, release := term.Login(ctx, nil)
	defer release()

	res, err := f.Struct()
	if err != nil {
		return auth.Session{}, err
	}

	sess, err := res.Session()
	if err != nil {
		return auth.Session{}, err
	}

	return auth.Session(sess).Clone(), nil
}
