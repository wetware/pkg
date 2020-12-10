package core

import (
	capnp "zombiezen.com/go/capnproto2"

	"github.com/wetware/ww/internal/mem"
	ww "github.com/wetware/ww/pkg"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
	memutil "github.com/wetware/ww/pkg/util/mem"
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
type Path struct{ mem.Any }

// NewPath .
func NewPath(a capnp.Arena, p string) (Path, error) {
	any, err := memutil.Alloc(a)
	if err == nil {
		err = any.SetPath(p)
	}

	return Path{any}, err
}

// Value returns the memory value
func (p Path) Value() mem.Any { return p.Any }

// Render the path into a parseable s-expression.
func (p Path) Render() (string, error) { return p.Path() }

// Parts returns split path for p
func (p Path) Parts() ([]string, error) {
	s, err := p.Path()
	if err != nil {
		return nil, err
	}

	return anchorpath.Parts(s), nil
}
