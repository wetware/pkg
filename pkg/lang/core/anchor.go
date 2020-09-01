package core

import (
	"context"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/pkg/errors"
	"github.com/spy16/sabre/runtime"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

var (
	_ runtime.Invokable = (*list)(nil)

	_ runtime.Value    = (*Path)(nil)
	_ apiValueProvider = (*Path)(nil)
)

/*
	Path
*/

// Path points to an anchor
type Path struct {
	v api.Value
}

// NewPath .
func NewPath(s string) (p Path, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(capnp.SingleSegment(nil)); err != nil {
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

func (p Path) String() string {
	str, err := p.v.Path()
	if err != nil {
		panic(err)
	}

	return str
}

// Parts of the path
func (p Path) Parts() []string {
	return anchorpath.Parts(p.String())
}

// Eval .
func (p Path) Eval(runtime.Runtime) (runtime.Value, error) {
	return p, nil
}

/*
	Anchor API
*/

// TODO(enhancement): replace with function or special form?
type list struct {
	ww.Anchor
}

func (root list) String() string {
	return "ls"
}

func (root list) Eval(r runtime.Runtime) (runtime.Value, error) {
	return root, nil
}

func (root list) Invoke(r runtime.Runtime, args ...runtime.Value) (runtime.Value, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("expected 1 argument, got %d", len(args))
	}

	p, ok := args[0].(Path)
	if !ok {
		return nil, errors.New("argument 0 must by of type Path")
	}

	as, err := root.Walk(context.Background(), p.Parts()).Ls(context.Background())
	if err != nil {
		return nil, err
	}

	b, err := NewVectorBuilder(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		/*
			TODO(performance):  we're effectively throwing away the anchor, here.

			Most of the time, the anchors retrieve by a call to `ls` will be used in a
			subsequent call.  How can we avoid the extra round-trip to retrieve them?

			Options include:

			- caching anchors at the rpc.Terminal level
			- binding the Anchor to the Path object (i.e.: caching at the core.Path level)
		*/

		if p, err = NewPath(a.String()); err != nil {
			return nil, err
		}

		if err = b.Conj(p); err != nil {
			return nil, err
		}
	}

	return b.Vector()
}
