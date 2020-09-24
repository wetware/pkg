// Package proc provides a plugin architecture for Wetware proceses.
package proc

import (
	"context"
	"errors"

	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

// import (
// 	"context"

// 	"github.com/pkg/errors"
// 	"github.com/spy16/parens"
// 	"github.com/wetware/ww/internal/api"
// 	ww "github.com/wetware/ww/pkg"
// 	"github.com/wetware/ww/pkg/mem"
// 	capnp "zombiezen.com/go/capnproto2"
// )

var (
	// 	registry = map[string]SpecParser{}

	// 	// ErrNotFound is returned by ParseSpec when the the requested
	// 	// parser could not be found.
	// 	ErrNotFound = errors.New("not found")

	_ ww.Proc = (*Proc)(nil)
	// _ ww.ProcSpec = (*goroutineSpec)(nil)
)

// Spawn configures a process based on the supplied arguments and then starts it.
func Spawn(env *parens.Env, args ...ww.Any) (ww.Proc, error) {

	// TODO(YOU ARE HERE)

	return Proc{}, errors.New("proc.Spawn() NOT IMPLEMENTED")
}

// Proc is a generic asynchronous process
type Proc struct{ mem.Value }

// New process
func New(a capnp.Arena, p api.Proc) (Proc, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetProc(p)
	}

	return Proc{val}, err
}

// SExpr returns a valid s-expression for proc.
func (p Proc) SExpr() (string, error) {
	return "TODO:  sexpr for proc", nil
}

// Wait for the process to terminate.  If ctx is cancelled, Wait returns.
func (p Proc) Wait(ctx context.Context) error {
	_, err := p.Raw.Proc().Wait(ctx, func(api.Proc_wait_Params) error { return nil }).Struct()
	return err
}

// // SpecParser can produce a process specification from
// // a sequence of arguments.
// type SpecParser func(parens.Seq) (ww.ProcSpec, error)

// // Register a SpecParser.  Duplicate names will be overwritten.
// func Register(name string, p SpecParser) {
// 	if len(name) == 0 {
// 		panic("process name cannot be empty")
// 	}

// 	if name[0] != ':' {
// 		panic("process names must start with ':'")
// 	}

// 	registry[name] = p
// }

// // ParseSpec a stream of arguments into a process specification.
// // The name string must be a keyword-formatted string (e.g. ":unix").
// func ParseSpec(name string, args parens.Seq) (ww.ProcSpec, error) {
// 	p, ok := registry[name]
// 	if !ok {
// 		return nil, ErrNotFound
// 	}

// 	return p(args)
// }

// type keywordArgs map[string]ww.Any

// func (kw keywordArgs) Get(any ww.Any) (ww.Any, bool, error) {
// 	// keys must be Keyword types
// 	if any.MemVal().Type() != api.Value_Which_keyword {
// 		return nil, false, nil
// 	}

// 	key, err := any.SExpr()
// 	if err != nil {
// 		return nil, false, err
// 	}

// 	val, ok := kw[key]
// 	return val, ok, nil
// }

// func seqToKwargs(seq parens.Seq) (keywordArgs, error) {
// 	n, err := seq.Count()
// 	if err != nil {
// 		return nil, err
// 	}

// 	if n == 0 {
// 		return nil, nil
// 	}

// 	if n%2 == 1 {
// 		return nil, errors.New("odd number of arguments")
// 	}

// 	n = 0
// 	var key string
// 	var m map[string]ww.Any

// 	return m, parens.ForEach(seq, func(item parens.Any) (halt bool, err error) {
// 		if n%2 == 0 {
// 			if key, err = item.SExpr(); err != nil {
// 				return
// 			}

// 			if len(key) == 0 || key[0] != ':' {
// 				return false, errors.New("key must be of type 'Keyword'")
// 			}
// 		} else {
// 			m[key] = item.(ww.Any)
// 		}

// 		n++
// 		return
// 	})
// }
