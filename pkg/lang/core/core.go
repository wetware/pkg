// Package core contains built-ins for the wetware language.
package core

import (
	"github.com/spy16/sabre/reader"
	"github.com/spy16/sabre/runtime"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang"
)

// RegisterMacros adds required language features to the reader.
func RegisterMacros(rd *reader.Reader) {
	for _, m := range []macroSpec{
		macro('/', false, pathMacro()),
		// macro('∂', false, fooMacro()),
	} {
		rd.SetMacro(m.Init, m.IsDispatch, m.Macro)
	}
}

// Bind registers core functions into the given scope.
func Bind(ww *lang.Ww, root ww.Anchor) error {
	return doUntilErr(ww,
		registerCore(root))
}

func registerCore(root ww.Anchor) func(*lang.Ww) error {
	return func(ww *lang.Ww) error {
		return registerList(ww, []mapEntry{
			// logical constants
			entry("true", runtime.Bool(true),
				"Represents logical true",
			),
			entry("false", runtime.Bool(false),
				"Represents logical false",
			),
			entry("nil", runtime.Nil{},
				"Represents logical false. Same as false",
			),

			// anchor API
			entry("ls", list{root},
				"(ls <path>)",
				"list an anchor path"),
		})
	}
}

func doUntilErr(ww *lang.Ww, fns ...func(*lang.Ww) error) error {
	for _, fn := range fns {
		if err := fn(ww); err != nil {
			return err
		}
	}

	return nil
}

func registerList(ww *lang.Ww, entries []mapEntry) error {
	for _, entry := range entries {
		if err := ww.BindDoc(entry.name, entry.val, entry.doc...); err != nil {
			return err
		}
	}

	return nil
}

func entry(name string, val runtime.Value, doc ...string) mapEntry {
	return mapEntry{
		name: name,
		val:  val,
		doc:  doc,
	}
}

type mapEntry struct {
	name string
	val  runtime.Value
	doc  []string
}

func macro(init rune, dispatch bool, m reader.Macro) macroSpec {
	return macroSpec{
		Init:       init,
		IsDispatch: dispatch,
		Macro:      m,
	}
}

type macroSpec struct {
	Init       rune
	IsDispatch bool
	Macro      reader.Macro
}
