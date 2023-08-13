package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/guest/system"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := system.Boot[host.Host](ctx)
	if err != nil {
		dief("boot: %w", err)
	}

	view, release := host.View(ctx)
	defer release()

	it, release := view.Iter(ctx, query())
	defer release()

	for r := it.Next(); r != nil; r = it.Next() {
		render(r)
	}

	die(it.Err())
}

func die(err error) {
	if err == nil {
		os.Exit(0)
	}

	fmt.Fprintln(os.Stdout, err)
	os.Exit(1)
}

func dief(format string, args ...any) {
	die(fmt.Errorf(format, args...))
}

func query() view.Query {
	return view.NewQuery(view.All())
}

func render(r routing.Record) {
	fmt.Fprintf(os.Stdout, "/%s\n", r.Server())
}
