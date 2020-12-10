package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wetware/ww/internal/mem"
	ww "github.com/wetware/ww/pkg"
	memutil "github.com/wetware/ww/pkg/util/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	errNoMatch = errors.New("does not match")
)

// A CallTarget is an implementation of a specific call signature for Fn.
type CallTarget struct {
	Name     string
	Param    []string
	Variadic bool
	Body     []ww.Any
}

// Fn is a multi-arity function or macro.
type Fn struct{ mem.Any }

// Value returns the memory value
func (fn Fn) Value() mem.Any { return fn.Any }

// Render a human-readable representation of the function
func (fn Fn) Render() (string, error) {
	// TODO:  more detail in rendering.
	return fn.Name()

	// name, err := fn.Name()
	// if err != nil {
	// 	return "", err
	// }

	// return fmt.Sprintf("func %s(%v)", name, strings.Join(args, ", ")), nil
}

// Macro returns true if the function is a macro.
func (fn Fn) Macro() bool {
	raw, err := fn.Fn()
	if err != nil {
		panic(err)
	}

	return raw.Macro()
}

// Name of the function.
func (fn Fn) Name() (string, error) {
	raw, err := fn.Fn()
	if err != nil {
		return "", err
	}

	if raw.Which() == mem.Fn_Which_lambda {
		return "λ", nil
	}

	return raw.Name()
}

// Match the arguments to the appropriate call signature.
func (fn Fn) Match(nargs int) (CallTarget, error) {
	raw, err := fn.Fn()
	if err != nil {
		return CallTarget{}, err
	}

	fs, err := raw.Funcs()
	if err != nil {
		return CallTarget{}, err
	}

	var ct CallTarget
	if ct.Name, err = fn.Name(); err != nil {
		return CallTarget{}, err
	}

	var a funcAnalyzer
	for i := 0; i < fs.Len(); i++ {
		a.f = fs.At(i)
		if ok, err := a.matchArity(nargs); err != nil {
			return CallTarget{}, err
		} else if !ok {
			continue
		}

		if ct.Param, err = a.params(); err != nil {
			return CallTarget{}, err
		}

		if ct.Body, err = a.body(); err != nil {
			return CallTarget{}, err
		}

		ct.Variadic = a.variadic()
		return ct, nil
	}

	return CallTarget{}, fmt.Errorf("%w (%d) to '%s'", ErrArity, nargs, ct.Name)
}

// FuncBuilder is a factory type for Fn.
type FuncBuilder struct {
	any    mem.Any
	fn     mem.Fn
	sigs   []callSignature
	stages []func() error
}

// Start building a function.
func (b *FuncBuilder) Start(a capnp.Arena) {
	b.stages = make([]func() error, 0, 8)
	b.sigs = b.sigs[:0]

	b.addStage(func() (err error) {
		if b.any, err = memutil.Alloc(a); err != nil {
			return fmt.Errorf("alloc value: %w", err)
		}

		if b.fn, err = b.any.NewFn(); err != nil {
			return fmt.Errorf("alloc fn: %w", err)
		}

		return nil
	})
}

// SetMacro sets the macro flag.
func (b *FuncBuilder) SetMacro(macro bool) {
	b.addStage(func() error {
		b.fn.SetMacro(macro)
		return nil
	})
}

// SetName assigns a name to the function.
func (b *FuncBuilder) SetName(name string) {
	b.addStage(func() error {
		if name == "" {
			b.fn.SetLambda()
			return nil
		}

		if err := b.fn.SetName(name); err != nil {
			return fmt.Errorf("set name: %w", err)
		}

		return nil
	})
}

// Commit flushes any buffers and returns the constructed function.
// After a call to Commit(), users must call Start() before reusing b.
func (b *FuncBuilder) Commit() (Fn, error) {
	for _, stage := range append(b.stages, b.setFuncs) {
		if err := stage(); err != nil {
			return Fn{}, err
		}
	}

	return Fn{Any: b.any}, nil
}

// AddSeq parses the sequence `([<params>*] <body>*)` into a call target.
func (b *FuncBuilder) AddSeq(seq Seq) {
	sig, err := ToSlice(seq)
	if err != nil {
		b.addStage(func() error { return err })
	}

	b.AddTarget(sig[0], sig[1:])
}

// AddTarget parses the call signature `[<params>*] <body>*` into a call target.
func (b *FuncBuilder) AddTarget(args ww.Any, body []ww.Any) {
	b.addStage(func() error {
		if any := args.Value(); args.Value().Which() != mem.Any_Which_vector {
			return Error{
				Cause:   errors.New("invalid call signature"),
				Message: fmt.Sprintf("args must be Vector, not '%s'", any.Which()),
			}
		}

		if body == nil {
			return Error{
				Cause:   errors.New("invalid call signature"),
				Message: "empty body",
			}
		}

		ps, variadic, err := b.readParams(args.(Vector))
		if err != nil {
			return err
		}

		b.sigs = append(b.sigs, callSignature{
			Params:   ps,
			Variadic: variadic,
			Body:     body,
		})
		return nil
	})
}

func (b *FuncBuilder) addStage(fn func() error) { b.stages = append(b.stages, fn) }

func (b *FuncBuilder) setFuncs() error {
	if len(b.sigs) == 0 {
		return errors.New("no call signatures")
	}

	fs, err := b.fn.NewFuncs(int32(len(b.sigs)))
	if err != nil {
		return err
	}

	for i, sig := range b.sigs {
		f := fs.At(i)
		if err = sig.Populate(f); err != nil {
			break
		}
	}

	return err
}

func (b *FuncBuilder) readParams(v Vector) ([]string, bool, error) {
	cnt, err := v.Count()
	if err != nil || cnt == 0 {
		return nil, false, err
	}

	ps := make([]string, cnt)

	for i := range ps {
		entry, err := v.EntryAt(i)
		if err != nil {
			return nil, false, err
		}

		if entry.Value().Which() != mem.Any_Which_symbol {
			return nil, false, fmt.Errorf("expected symbol, got %s", entry.Value().Which())
		}

		if ps[i], err = entry.Value().Symbol(); err != nil {
			return nil, false, err
		}
	}

	var variadic bool
	if last := ps[cnt-1]; strings.HasSuffix(last, "...") {
		ps[cnt-1] = last[:len(last)-3]
		variadic = true
	}

	return ps, variadic, nil
}

type callSignature struct {
	Params   []string
	Variadic bool
	Body     []ww.Any
}

func (sig callSignature) Populate(f mem.Fn_Func) (err error) {
	if err = sig.populateBody(f); err == nil {
		err = sig.populateParams(f)
	}

	return
}

func (sig callSignature) populateParams(f mem.Fn_Func) error {
	if sig.Params == nil {
		f.SetNilary()
		return nil
	}

	as, err := f.NewParams(int32(len(sig.Params)))
	if err != nil {
		return err
	}

	for i, s := range sig.Params {
		if err = as.Set(i, s); err != nil {
			break
		}
	}

	if sig.Variadic {
		f.SetVariadic(true)
	}

	return err
}

func (sig callSignature) populateBody(f mem.Fn_Func) error {
	bs, err := f.NewBody(int32(len(sig.Body)))
	if err != nil {
		return err
	}

	for i, any := range sig.Body {
		if err = bs.Set(i, any.Value()); err != nil {
			break
		}
	}

	return err
}

type funcAnalyzer struct {
	f      mem.Fn_Func
	ps     capnp.TextList
	nparam int
}

func (a *funcAnalyzer) matchArity(nargs int) (bool, error) {
	// if there are no params -> must have 0 args
	if !a.f.HasParams() {
		return nargs == 0, nil
	}

	var err error
	if a.ps, err = a.f.Params(); err != nil {
		return false, err
	}

	a.nparam = a.ps.Len()
	if a.variadic() {
		return nargs >= a.nparam-1, nil
	}

	return nargs == a.nparam, nil
}

func (a funcAnalyzer) variadic() bool { return a.f.Variadic() }

func (a funcAnalyzer) params() (ps []string, err error) {
	ps = make([]string, a.nparam)
	for i := range ps {
		if ps[i], err = a.ps.At(i); err != nil {
			break
		}
	}

	return
}

func (a funcAnalyzer) body() (forms []ww.Any, err error) {
	var vs mem.Any_List
	if vs, err = a.f.Body(); err != nil {
		return
	}

	forms = make([]ww.Any, vs.Len())
	for i := range forms {
		if forms[i], err = AsAny(vs.At(i)); err != nil {
			break
		}
	}

	return
}
