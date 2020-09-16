// Package lang contains the wetware language iplementation
package lang

import "github.com/spy16/parens"

// New returns a new root environment.
func New() *parens.Env {
	return parens.New(
	// TODO
	// parens.WithAnalyzer(...),
	// parens.WithGlobals(nil, globalMapFactory),
	)
}

// import (
// 	"strings"
// 	"sync"

// 	"github.com/spy16/sabre/reader"
// 	"github.com/spy16/sabre/runtime"
// 	ww "github.com/wetware/ww/pkg"
// 	"github.com/wetware/ww/pkg/lang/core"
// 	bindutil "github.com/wetware/ww/pkg/lang/util/bind"
// )

// var _ runtime.Runtime = (*Ww)(nil)

// // Ww language runtime
// type Ww struct {
// 	mu     sync.RWMutex
// 	parent *Ww
// 	scope  map[string]scopeItem
// }

// // New language runtime
// func New(parent *Ww) *Ww {
// 	return &Ww{
// 		parent: parent,
// 		scope:  make(map[string]scopeItem),
// 	}
// }

// // Eval evaluates the form in this runtime. Runtime might customize the eval
// // rules for different values, but in most cases, eval will be dispatched to
// // Eval() method of value.
// func (ww *Ww) Eval(form runtime.Value) (runtime.Value, error) {
// 	if core.IsNil(form) {
// 		return core.Nil{}, nil
// 	}

// 	v, err := form.Eval(ww)
// 	if err != nil {
// 		e := runtime.NewErr(false, getPosition(form), err)
// 		e.Form = form
// 		return nil, e
// 	}

// 	if v == nil {
// 		return core.Nil{}, nil
// 	}

// 	return v, nil
// }

// // Bind binds the value to the symbol. Returns error if the symbol contains
// // invalid character or the bind fails for some other reasons.
// func (ww *Ww) Bind(symbol string, v runtime.Value) error {
// 	return ww.BindDoc(symbol, v, "")
// }

// // BindDoc binds a value and a docstring to a symbol.  See Bind.
// func (ww *Ww) BindDoc(symbol string, v runtime.Value, doc ...string) error {
// 	ww.mu.Lock()
// 	defer ww.mu.Unlock()

// 	ww.scope[symbol] = scopeItem{
// 		doc: strings.TrimSpace(strings.Join(doc, "\n")),
// 		val: v,
// 	}

// 	return nil
// }

// // Resolve returns the value bound for the the given symbol. Resolve returns
// // ErrNotFound if the symbol has no binding.
// func (ww *Ww) Resolve(symbol string) (runtime.Value, error) {
// 	ww.mu.RLock()
// 	defer ww.mu.RUnlock()

// 	if item, ok := ww.scope[symbol]; ok {
// 		return item.val, nil
// 	}

// 	return nil, runtime.ErrNotFound
// }

// // Parent returns the parent of this environment. If returned value is nil,
// // it is the root Runtime.
// func (ww *Ww) Parent() runtime.Runtime {
// 	return ww.parent
// }

// // Doc returns the docstring for the specified symbol.
// func (ww *Ww) Doc(symbol string) string {
// 	ww.mu.RLock()
// 	defer ww.mu.RUnlock()

// 	if v, ok := ww.scope[symbol]; ok {
// 		return v.doc
// 	}

// 	if ww.parent != nil {
// 		return ww.parent.Doc(symbol)
// 	}

// 	return ""
// }

// // BindAll registers core functions into the given scope.
// func BindAll(r runtime.Runtime, root ww.Anchor) error {
// 	return bindutil.BindAll(r,
// 		core.Bind(root))
// }

// type scopeItem struct {
// 	doc string
// 	val runtime.Value
// }

// func getPosition(form runtime.Value) (p runtime.Position) {
// 	if f, ok := form.(interface{ GetPos() (string, int, int) }); ok {
// 		p.File, p.Line, p.Column = f.GetPos()
// 	}

// 	return
// }

// func macro(init rune, dispatch bool, m reader.Macro) macroSpec {
// 	return macroSpec{
// 		Init:       init,
// 		IsDispatch: dispatch,
// 		Macro:      m,
// 	}
// }

// type macroSpec struct {
// 	Init       rune
// 	IsDispatch bool
// 	Macro      reader.Macro
// }
