package lang

import (
	capnp "zombiezen.com/go/capnproto2"

	"github.com/spy16/parens"

	"github.com/wetware/ww/internal/api"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

var (
	_ parens.Any       = (*Path)(nil)
	_ apiValueProvider = (*Path)(nil)
)

// Path points to an anchor
type Path struct {
	v api.Value
}

// NewPath .
func NewPath(a capnp.Arena, s string) (p Path, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if p.v, err = api.NewRootValue(seg); err == nil {
		err = p.v.SetPath(s)
	}

	return
}

// Value for Path type
func (p Path) Value() api.Value {
	return p.v
}

// SExpr returns a valid s-expression for path.
func (p Path) SExpr() (string, error) {
	return p.v.Path()
}

// Parts returns split path for p
func (p Path) Parts() ([]string, error) {
	s, err := p.v.Path()
	if err != nil {
		return nil, err
	}

	return anchorpath.Parts(s), nil
}

/*
	Anchor API
*/

// // TODO(enhancement): replace with function or special form?
// type list struct {
// 	ww.Anchor
// }

// func (root list) String() string {
// 	return "ls"
// }

// func (root list) Invoke(env *parens.Env, args ...parens.Any) (parens.Any, error) {
// 	if len(args) != 1 {
// 		return nil, errors.Errorf("expected 1 argument, got %d", len(args))
// 	}

// 	p, ok := args[0].(Path)
// 	if !ok {
// 		return nil, errors.New("argument 0 must by of type Path")
// 	}

// 	pstr, err := p.v.Path()
// 	if err != nil {
// 		return nil, err
// 	}

// 	as, err := root.Walk(context.Background(), anchorpath.Parts(pstr)).Ls(context.Background())
// 	if err != nil {
// 		return nil, err
// 	}

// 	b, err := NewVectorBuilder(capnp.SingleSegment(nil))
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, a := range as {
// 		/*
// 			TODO(performance):  we're effectively throwing away the anchor, here.

// 			Most of the time, the anchors retrieve by a call to `ls` will be used in a
// 			subsequent call.  How can we avoid the extra round-trip to retrieve them?

// 			Options include:

// 			- caching anchors at the rpc.Terminal level
// 			- binding the Anchor to the Path object (i.e.: caching at the core.Path level)
// 		*/

// 		if p, err = NewPath(capnp.SingleSegment(nil), a.String()); err != nil {
// 			return nil, err
// 		}

// 		if err = b.Conj(p); err != nil {
// 			return nil, err
// 		}
// 	}

// 	return b.Vector()
// }
