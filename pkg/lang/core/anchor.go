package core

import (
	"context"
	"errors"
	"reflect"

	"github.com/spy16/sabre"
	"github.com/spy16/sabre/runtime"
	ww "github.com/wetware/ww/pkg"
)

var (
	_ runtime.Invokable = (*list)(nil)
)

/*
	Anchor API
*/

type list struct {
	ww.Anchor
}

func (root list) String() string {
	return "ls"
}

func (root list) Eval(r runtime.Runtime) (runtime.Value, error) {
	return root, nil
}

func (root list) Invoke(r runtime.Runtime, args ...runtime.Value) (runtime.Value, error) {
	p, ok := args[0].(Path)
	if !ok {
		return nil, errors.New("argument 0 must by of type Path")
	}

	as, err := root.Walk(context.Background(), p).
		Ls(context.Background())
	if err != nil {
		return nil, err
	}

	// TODO:  replace Any with Vector or Set implementation.
	return sabre.Any{
		V: reflect.ValueOf(as),
	}, nil
}
