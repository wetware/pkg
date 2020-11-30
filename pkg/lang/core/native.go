package core

import (
	"fmt"
	"reflect"
	"strings"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	anyType = reflect.TypeOf((*ww.Any)(nil)).Elem()
	errType = reflect.TypeOf((*error)(nil)).Elem()

	_ Invokable = (*funcWrapper)(nil)
)

// Func converts the given Go func to a ww.Any that is guaranteed
// to satisfy Invokable.
func Func(name string, v interface{}) (ww.Any, error) {
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	if rt.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s is not a func", rt)
	}

	minArgs := rt.NumIn()
	if rt.IsVariadic() {
		minArgs = minArgs - 1
	}

	lastOutIdx := rt.NumOut() - 1
	returnsErr := lastOutIdx >= 0 && rt.Out(lastOutIdx) == errType
	if returnsErr {
		lastOutIdx-- // ignore error value from return values
	}

	for i := 0; i <= lastOutIdx; i++ {
		if !rt.Out(i).AssignableTo(anyType) {
			return nil, fmt.Errorf("return value %d (%s) not assignable to %s",
				i, rt.Out(i), anyType)
		}
	}

	sym, err := NewSymbol(capnp.SingleSegment(nil), name)
	if err != nil {
		return nil, err
	}

	return &funcWrapper{
		sym:        sym,
		rv:         rv,
		rt:         rt,
		minArgs:    minArgs,
		returnsErr: returnsErr,
		lastOutIdx: lastOutIdx,
	}, nil
}

type funcWrapper struct {
	sym        Symbol
	rv         reflect.Value
	rt         reflect.Type
	minArgs    int
	returnsErr bool
	lastOutIdx int
}

func (fw *funcWrapper) MemVal() mem.Value { return fw.sym.MemVal() }

func (fw *funcWrapper) Invoke(args ...ww.Any) (ww.Any, error) {
	// allocate argument slice.
	argCount := len(args)
	argVals := make([]reflect.Value, argCount, argCount)

	// populate reflect.Value version of each argument.
	for i, arg := range args {
		argVals[i] = reflect.ValueOf(arg)
	}

	// verify number of args match the required function parameters.
	if err := fw.checkArgCount(len(argVals)); err != nil {
		return nil, err
	}

	if err := fw.convertTypes(argVals...); err != nil {
		return nil, err
	}

	return fw.wrapReturns(fw.rv.Call(argVals)...)
}

func (fw *funcWrapper) String() string {
	args := fw.argNames()
	if fw.rt.IsVariadic() {
		args[len(args)-1] = "..." + args[len(args)-1]
	}

	for i, arg := range args {
		args[i] = fmt.Sprintf("arg%d %s", i, arg)
	}

	name, err := fw.sym.Symbol()
	if err != nil {
		panic(err) // unreachable since call to Func() succeeded.
	}

	return fmt.Sprintf("func %s(%v)", name, strings.Join(args, ", "))
}

func (fw *funcWrapper) argNames() []string {
	cleanArgName := func(t reflect.Type) string {
		return strings.Replace(t.String(), "lang.", "", -1)
	}

	var argNames []string

	i := 0
	for ; i < fw.minArgs; i++ {
		argNames = append(argNames, cleanArgName(fw.rt.In(i)))
	}

	if fw.rt.IsVariadic() {
		argNames = append(argNames, cleanArgName(fw.rt.In(i).Elem()))
	}

	return argNames
}

func (fw *funcWrapper) convertTypes(args ...reflect.Value) error {
	lastArgIdx := fw.rt.NumIn() - 1
	isVariadic := fw.rt.IsVariadic()

	for i := 0; i < fw.rt.NumIn(); i++ {
		if i == lastArgIdx && isVariadic {
			c, err := convertArgsTo(fw.rt.In(i).Elem(), args[i:]...)
			if err != nil {
				return err
			}
			copy(args[i:], c)
			break
		}

		c, err := convertArgsTo(fw.rt.In(i), args[i])
		if err != nil {
			return err
		}
		args[i] = c[0]
	}

	return nil
}

func (fw *funcWrapper) checkArgCount(count int) error {
	if count != fw.minArgs {
		if fw.rt.IsVariadic() && count < fw.minArgs {
			return fmt.Errorf(
				"call requires at-least %d argument(s), got %d",
				fw.minArgs, count,
			)
		}

		if !fw.rt.IsVariadic() {
			return fmt.Errorf(
				"call requires exactly %d argument(s), got %d",
				fw.minArgs, count,
			)
		}
	}

	return nil
}

func (fw *funcWrapper) wrapReturns(vals ...reflect.Value) (ww.Any, error) {
	if fw.rt.NumOut() == 0 {
		return Nil{}, nil
	}

	if fw.returnsErr {
		errIndex := fw.lastOutIdx + 1
		if !vals[errIndex].IsNil() {
			return nil, vals[errIndex].Interface().(error)
		}

		if fw.rt.NumOut() == 1 {
			return Nil{}, nil
		}
	}

	retValCount := len(vals[0 : fw.lastOutIdx+1])
	wrapped := make([]ww.Any, retValCount, retValCount)
	for i := 0; i < retValCount; i++ {
		wrapped[i] = vals[i].Interface().(ww.Any) // TODO(performance):  unsafe.Pointer?
	}

	if retValCount == 1 {
		return wrapped[0], nil
	}

	return NewList(capnp.SingleSegment(nil), wrapped...)
}

func convertArgsTo(expected reflect.Type, args ...reflect.Value) ([]reflect.Value, error) {
	converted := make([]reflect.Value, len(args), len(args))
	for i, arg := range args {
		actual := arg.Type()
		isAssignable := (actual == expected) ||
			actual.AssignableTo(expected) ||
			(expected.Kind() == reflect.Interface && actual.Implements(expected))
		if isAssignable {
			converted[i] = arg
		} else if actual.ConvertibleTo(expected) {
			converted[i] = arg.Convert(expected)
		} else {
			return args, fmt.Errorf(
				"value of type '%s' cannot be converted to '%s'",
				actual, expected,
			)
		}
	}

	return converted, nil
}
