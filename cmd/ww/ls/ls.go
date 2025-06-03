package ls

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/cmd/ww/cluster"
	"github.com/wetware/pkg/vat"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "ls",
		Action: list,
	}
}

func list(c *cli.Context) error {
	h, err := vat.DialP2P()
	if err != nil {
		return err
	}

	sess, close, err := cluster.BootstrapSession(c, h)
	defer close()
	if err != nil {
		return err
	}

	it, release := sess.View().Iter(c.Context, query(c))
	defer release()

	for r := it.Next(); r != nil; r = it.Next() {
		render(c, r)
	}

	return it.Err()
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, r routing.Record) {
	fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
}
