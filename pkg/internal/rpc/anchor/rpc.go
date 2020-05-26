package anchor

import (
	"context"

	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/rpc"
)

// Ls .
func Ls(ctx context.Context, t rpc.Terminal, d rpc.Dialer) ([]ww.Anchor, error) {
	h := &ls{Terminal: t}
	t.Call(ctx, d, h)
	return h.Resolve()
}

// Walk returns the anchor at the specified path.
//
// Walk panics if len(path) == 0.
func Walk(ctx context.Context, t rpc.Terminal, d rpc.Dialer, path []string) ww.Anchor {
	a := anchor{path: path}
	t.Call(ctx, rpc.DialString(path[0]), (*walk)(&a))
	return a
}
