package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	ww "github.com/wetware/pkg/wasm"
)

func main() {
	ctx := context.Background()
	client, closer, err := ww.BootstrapClient(ctx)
	defer closer.Close()
	if err != nil {
		panic(err)
	}

	host := host.Host(client)
	defer host.Release()

	view, release := host.View(context.Background())
	defer release()

	it, release := view.Iter(context.Background(), query())
	defer release()

	for r := it.Next(); r != nil; r = it.Next() {
		render(r)
	}
}

func query() view.Query {
	return view.NewQuery(view.All())
}

func render(r routing.Record) {
	fmt.Fprintf(os.Stdout, "/%s\n", r.Server())
}
