package bindutil

import "github.com/spy16/sabre/runtime"

type Binder interface {
	Bind(runtime.Runtime) error
}

type BinderFunc func(runtime.Runtime) error

func (f BinderFunc) Bind(r runtime.Runtime) error {
	return f(r)
}

func BindAll(r runtime.Runtime, bs ...Binder) error {
	for _, b := range bs {
		if err := b.Bind(r); err != nil {
			return err
		}
	}

	return nil
}

func BindList(r runtime.Runtime, entries []Binding) (err error) {
	for _, entry := range entries {
		if b, ok := r.(docBinder); ok {
			err = b.BindDoc(entry.name, entry.val, entry.doc...)
		} else {
			err = r.Bind(entry.name, entry.val)
		}

		if err != nil {
			break
		}
	}

	return
}

func Bind(name string, val runtime.Value, doc ...string) Binding {
	return Binding{
		name: name,
		val:  val,
		doc:  doc,
	}
}

type Binding struct {
	name string
	val  runtime.Value
	doc  []string
}

type docBinder interface {
	BindDoc(string, runtime.Value, ...string) error
}
