package ps

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/urfave/cli/v2"

	proc_api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/cmd/ww/cluster"
	"github.com/wetware/pkg/util/proto"
	"github.com/wetware/pkg/vat"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "ps",
		Usage: "list processes running in the cluster",
		Action: func(c *cli.Context) error {
			// Get a session.
			h, err := vat.DialP2P()
			if err != nil {
				return err
			}
			sess, close, err := cluster.BootstrapSession(c, h)
			defer close()
			if err != nil {
				return err
			}

			// Cluster view.
			view := auth.Session(sess).View()
			it, release := view.Iter(c.Context, query(c))
			defer release()

			// The table writer will format the output columns.
			tw := new(tabwriter.Writer)
			tw.Init(c.App.Writer, 8, 8, 0, '\t', 0)
			defer tw.Flush()

			// Used to dial each host in the view.
			d := vat.Dialer{
				Host:    h,
				Account: auth.SignerFromHost(h),
			}
			for r := it.Next(); r != nil; r = it.Next() {
				peer := r.Peer()
				if peer == h.ID() {
					continue // skip self, as it has no executor.
				}
				// Get a new session from the host.
				nsess, err := d.Dial(
					c.Context,
					h.Peerstore().PeerInfo(peer),
					proto.Namespace(c.String("ns"))...)
				if err != nil {
					fmt.Fprintln(c.App.ErrWriter, err.Error())
					continue
				}

				// Render the executor.
				e := nsess.Exec()
				defer e.Release()
				defer release()
				renderExec(c, tw, r, e)
			}

			return it.Err()
		},
	}
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

// Call e.Ps and render the output.
func renderExec(c *cli.Context, tw *tabwriter.Writer, r routing.Record, e csp.Executor) {
	procs, release, err := e.Ps(c.Context)
	defer release()
	if err != nil {
		fmt.Fprintln(tw, err.Error())
		return
	}

	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t\n", "Executor", "PID", "PPID", "Creation", "CID", "Args")
	for _, proc := range procs {
		renderInfo(c, tw, proc, r.Server().String())
	}
}

// Render a process running in the executor.
func renderInfo(c *cli.Context, tw *tabwriter.Writer, info proc_api.Info, peer string) {
	// Extract process information.
	b, err := info.Cid()
	if err != nil {
		fmt.Fprintln(c.App.ErrWriter, err.Error())
		return
	}
	_, cid, _ := cid.CidFromBytes(b)
	al, err := info.Argv()
	if err != nil {
		fmt.Fprintln(c.App.ErrWriter, err.Error())
		return
	}
	argv := make([]string, al.Len())
	for i := 0; i < al.Len(); i++ {
		arg, err := al.At(i)
		if err != nil {
			fmt.Fprintln(c.App.ErrWriter, err.Error())
			return
		}
		argv[i] = arg
	}

	// Actual rendering.
	fmt.Fprintf(tw, "%s\t%d\t%d\t%s\t%s\t%s\n",
		peer,
		info.Pid(),
		info.Ppid(),
		time.UnixMilli(int64(info.Time())).Format(time.UnixDate),
		cid.Encode(multibase.MustNewEncoder(multibase.Base58BTC)),
		argv,
	)
}
