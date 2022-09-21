package client

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/ww/pkg/anchor"
)

func query(c *cli.Context) cluster.Query {
	path, args := parsePath(c.Args().Slice())
	if path.Err() != nil {
		return failuref("path: %w", path.Err())
	}

	selector, args := parseSelector(args)
	cs, args := parseConstraints(args)

	if args.Len() > 0 {
		return failuref("unexpected token: %s", args.Head())
	}

	return cluster.NewQuery(selector, cs...)
}

func parsePath(args arguments) (anchor.Path, arguments) {
	switch head(args) {
	case "", "WHERE", "FROM":
		// No path specified; default to root.
		return anchor.NewPath("/"), args

	default:
		// Paths are case-sensitive; use args.Head() directly
		return anchor.NewPath(args.Head()), args.Tail()
	}
}

func parseSelector(args arguments) (cluster.Selector, arguments) {
	switch head(args) {
	case "WHERE":
		return parseMatch(args.Tail())

	// case "FROM":
	// 	return parseRange(args.Tail())

	default:
		return cluster.All(), args
	}
}

func parseMatch(args arguments) (cluster.Selector, arguments) {
	index, args, err := parseIndexExpr(args)
	if err != nil {
		return sfailf("match: %w", err), nil
	}

	return cluster.Match(index), args.Tail()
}

func sfailf(format string, args ...any) cluster.Selector {
	return func(cluster.SelectorStruct) error {
		return fmt.Errorf(format, args...)
	}
}

// func parseRange(args arguments) (cluster.Selector, arguments) {
// 	return func(cluster.SelectorStruct) error {
// 		return errors.New("parse: MATCH: NOT IMPLEMENTED")
// 	}, nil
// }

func parseConstraints(args arguments) ([]cluster.Constraint, arguments) {
	return nil, args
}

func failuref(format string, args ...any) cluster.Query {
	return func(cluster.QueryParams) error {
		return fmt.Errorf(format, args...)
	}
}

func parseIndexExpr(args arguments) (routing.Index, arguments, error) {
	expr := indexExpr(args)
	if len(expr) != 2 {
		return nil, nil, fmt.Errorf("invalid index expression: %s", args.Head())
	}

	switch expr.String() {
	case "id", "host":
		return index(expr), args.Tail(), nil

	default:
		return index{"meta", args.Head()}, args.Tail(), nil
	}
}

func isPrefix(s string) bool {
	return strings.HasSuffix(s, "_prefix")
}

type index []string

func indexExpr(args arguments) index {
	return strings.SplitN(args.Head(), "=", 2)
}

func (ix index) Prefix() bool {
	return isPrefix(ix[0])
}

func (ix index) String() string {
	return strings.TrimSuffix(ix[0], "_prefix")
}

func (ix index) PeerBytes() ([]byte, error) {
	if !strings.HasPrefix(ix[0], "id") {
		panic("not a peer index")
	}

	return []byte(ix[1]), nil
}

func (ix index) HostBytes() ([]byte, error) {
	if !strings.HasPrefix(ix[0], "host") {
		panic("not a host index")
	}

	return []byte(ix[1]), nil
}

func (ix index) MetaField() (routing.MetaField, error) {
	return routing.ParseField(ix[1])
}

func head(args arguments) string {
	return strings.ToUpper(args.Head())
}

type arguments []string

func (as arguments) Len() int { return len(as) }

func (as arguments) Get(n int) (arg string) {
	if as.Len() > n {
		arg = as[n]
	}

	return
}

func (as arguments) Head() string {
	return as.Get(0)
}

func (as arguments) Tail() (tail []string) {
	if as.Len() >= 2 {
		tail = []string(as)[1:]
	}

	return
}
