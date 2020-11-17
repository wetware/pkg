package core

import (
	ww "github.com/wetware/ww/pkg"
)

// Invokable represents a value that can be invoked as a function.
type Invokable interface {
	// Invoke is called if this value appears as the first argument of
	// invocation form (i.e., list).
	Invoke(args ...ww.Any) (ww.Any, error)
}

// List is a persistent, singly-linked list with fast insertions/pops to its head.
type List interface {
	ww.Any
	Seq
	Count() (int, error)
	Cons(any ww.Any) (List, error)
}
