package core

import (
	capnp "zombiezen.com/go/capnproto2"

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
func NewPath(a capnp.Arena, p string) (Path, error) {
	mv, err := mem.NewValue(a)
	if err == nil {
		err = mv.MemVal().SetPath(p)
	}

	return Path{mv}, err
}

// Render the path into a parseable s-expression.
func (p Path) Render() (string, error) { return p.MemVal().Path() }

// Parts returns split path for p
func (p Path) Parts() ([]string, error) {
	s, err := p.MemVal().Path()
	if err != nil {
		return nil, err
	}

	return anchorpath.Parts(s), nil
}
