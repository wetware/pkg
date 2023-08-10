// Package bounded implements generic bounded types for validation.
package bounded

// Type is a bounded type.  If err == nil, then T
// is valid.
type Type[T any] struct {
	t   T
	err error
}

func (t Type[T]) Maybe() (T, error) {
	return t.t, t.err
}

type Validator[T any] func(t T) Type[T]

// Bind returns a new instance of the bounded type that
// has been validated by f.
func (t Type[T]) Bind(f Validator[T]) Type[T] {
	if t.err == nil {
		return f(t.t)
	}

	return t
}

func Failure[T any](err error) Type[T] {
	return Type[T]{err: err}
}

func Value[T any](t T) Type[T] {
	return Type[T]{t: t}
}
