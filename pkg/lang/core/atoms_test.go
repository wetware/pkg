package core_test

import (
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func mustSymbol(s string) core.Symbol {
	sym, err := core.NewSymbol(capnp.SingleSegment(nil), s)
	if err != nil {
		panic(err)
	}

	return sym
}

func mustKeyword(s string) core.Keyword {
	kw, err := core.NewKeyword(capnp.SingleSegment(nil), s)
	if err != nil {
		panic(err)
	}

	return kw
}

func mustString(s string) core.String {
	str, err := core.NewString(capnp.SingleSegment(nil), s)
	if err != nil {
		panic(err)
	}

	return str
}

func mustChar(r rune) core.Char {
	c, err := core.NewChar(capnp.SingleSegment(nil), r)
	if err != nil {
		panic(err)
	}

	return c
}

func mustRender(v ww.Any) string {
	sexpr, err := core.Render(v)
	if err != nil {
		panic(err)
	}

	return sexpr
}
