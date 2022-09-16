package client

import (
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/urfave/cli/v2"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/routing"
)

func list() *cli.Command {
	return &cli.Command{
		Name:  "ls",
		Usage: "list anchor elements",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Usage:   "print results as json",
				EnvVars: []string{"WW_FMT_JSON"},
			},
		},
		Before: setup(),
		Action: ls(),
		After:  teardown(),
	}
}

func ls() cli.ActionFunc {
	return func(c *cli.Context) error {
		view, release := node.View(c.Context)
		defer release()

		it, release := view.Iter(c.Context, cluster.All())
		defer release()

		return render(it, formatter(c))
	}
}

func render(it cluster.Iterator, consume func(routing.Record) error) error {
	for record := it.Next(); record != nil; record = it.Next() {
		if err := consume(record); err != nil {
			return err
		}
	}

	return it.Err()
}

func formatter(c *cli.Context) func(routing.Record) error {
	if c.Bool("json") {
		return jsonFormatter(c)
	}

	return textFormatter(c)
}

func jsonFormatter(c *cli.Context) func(routing.Record) error {
	enc := json.NewEncoder(c.App.Writer)

	return func(r routing.Record) error {
		rec, err := asJSON(r)
		if err == nil {
			err = enc.Encode(rec)
		}
		return err
	}
}

type jsonRecord struct {
	Peer     peer.ID           `json:"peer"`
	Seq      uint64            `json:"seq"`
	Instance routing.ID        `json:"instance"`
	Host     string            `json:"host,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

func asJSON(r routing.Record) (rec jsonRecord, err error) {
	rec.Peer = r.Peer()
	rec.Seq = r.Seq()
	rec.Instance = r.Instance()

	if rec.Host, err = r.Host(); err != nil {
		return
	}

	var meta routing.Meta
	if meta, err = r.Meta(); err != nil {
		return
	}

	var field routing.Field
	for i := 0; i < meta.Len(); i++ {
		if field, err = meta.At(i); err != nil {
			break
		}

		if rec.Meta == nil {
			rec.Meta = make(map[string]string)
		}

		rec.Meta[field.Key] = field.Value
	}

	return
}

func textFormatter(c *cli.Context) func(routing.Record) error {
	return func(r routing.Record) error {
		_, err := fmt.Fprintf(c.App.Writer, "/%s\n", r.Peer())
		return err
	}
}
