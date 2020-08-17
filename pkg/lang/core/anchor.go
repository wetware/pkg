package core

import (
	"context"
	"errors"
	"reflect"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/spy16/sabre"
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
	p, ok := args[0].(Path)
	if !ok {
		return nil, errors.New("argument 0 must by of type Path")
	}

	as, err := root.Walk(context.Background(), p.Parts()).Ls(context.Background())
	if err != nil {
		return nil, err
	}

	// TODO:  replace Any with Vector or Set implementation.
	return sabre.Any{
		V: reflect.ValueOf(as),
	}, nil
}
