package main

import (
	"context"
	"fmt"
	"os"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/guest/system"
	"golang.org/x/exp/slog"
)

var (
	ctx = context.Background()
	log = slog.Default()
)

func main() {
	host, release := system.Boot[host.Host](ctx)
	defer release()

	// TODO(performance):  remove this once we've addressed any
	// promise pipelining bugs in Cap'n Proto.
	if err := capnp.Client(host).Resolve(ctx); err != nil {
		die(err)
	}

	view, release := host.View(ctx)
	defer release()

	// TODO(performance):  remove this once we've addressed any
	// promise pipelining bugs in Cap'n Proto.
	if err := capnp.Client(view).Resolve(ctx); err != nil {
		die(err)
	}

	it, release := view.Iter(ctx, query())
	defer release()

	die(render(it))
}

func die(err error) {
	if err == nil {
		os.Exit(0)
	}

	fmt.Fprintln(os.Stdout, err)
	os.Exit(1)
}

func query() view.Query {
	return view.NewQuery(view.All())
}

func render(it view.Iterator) error {
	for r := it.Next(); r != nil; r = it.Next() {
		fmt.Fprintf(os.Stdout, "/%s\n", r.Server())
	}

	return it.Err()
}
