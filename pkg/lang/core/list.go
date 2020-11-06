package core

import (
	"github.com/spy16/slurp/core"
	ww "github.com/wetware/ww/pkg"
)

// Invokable represents a value that can be invoked for result.
type Invokable = core.Invokable

// List is a persistent, singly-linked list with fast insertions/pops to its head.
type List interface {
	ww.Any
	Seq
	Count() (int, error)
	Cons(any ww.Any) (List, error)
}
