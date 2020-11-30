package core

import (
	capnp "zombiezen.com/go/capnproto2"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

var (
	// RootPath for Anchor hierarchy.
	RootPath Path

	_ ww.Any = (*Path)(nil)
)

func init() {
	var err error
	if RootPath, err = NewPath(capnp.SingleSegment(nil), "/"); err != nil {
		panic(err)
	}
}

// Path points to an anchor
type Path struct{ mem.Value }

// NewPath .
func NewPath(a capnp.Arena, s string) (p Path, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if p.Raw, err = api.NewRootValue(seg); err == nil {
		err = p.Raw.SetPath(s)
	}

	return
}

// Render the path into a parseable s-expression.
func (p Path) Render() (string, error) { return p.Raw.Path() }

// Parts returns split path for p
func (p Path) Parts() ([]string, error) {
	s, err := p.Raw.Path()
	if err != nil {
		return nil, err
	}

	return anchorpath.Parts(s), nil
}
