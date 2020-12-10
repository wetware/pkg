package lang

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	anyType = reflect.TypeOf((*ww.Any)(nil)).Elem()
	errType = reflect.TypeOf((*error)(nil)).Elem()
	ivkType = reflect.TypeOf((*core.Invokable)(nil)).Elem()

	_ core.Invokable = (*funcWrapper)(nil)
)

// Func converts the given Go value into a Wetware native function.
// The resulting value is guaranteed to be invokable.
func Func(name string, v interface{}) (ww.Any, error) {
	rv := reflect.ValueOf(v)
	rt := rv.Type()

	if m, ok := rt.MethodByName("Invoke"); ok {
		rv = m.Func
		rt = rv.Type()
	}

	if rt.Kind() != reflect.Func {
		return nil, fmt.Errorf("cannot convert '%s' to func", reflect.TypeOf(v))
	}

	return newFuncWrapper(name, rv, rt)
}

func newFuncWrapper(name string, rv reflect.Value, rt reflect.Type) (*funcWrapper, error) {
	minArgs := rt.NumIn()
	if rt.IsVariadic() {
		minArgs = minArgs - 1
	}

	lastOutIdx := rt.NumOut() - 1
	returnsErr := lastOutIdx >= 0 && rt.Out(lastOutIdx) == errType
	if returnsErr {
		lastOutIdx-- // ignore error value from return values
	}

	adapters := make([]adapter, lastOutIdx+1)
	for i := range adapters {
		if out := rt.Out(i); !out.AssignableTo(anyType) {
			adapters[i] = adapterFor(out)
		}
	}

	sym, err := core.NewSymbol(capnp.SingleSegment(nil), name)
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
		adapters:   adapters,
	}, nil
}

type funcWrapper struct {
	sym        core.Symbol
	rv         reflect.Value
	rt         reflect.Type
	minArgs    int
	returnsErr bool
	lastOutIdx int
	adapters   []adapter
}

func (fw *funcWrapper) MemVal() api.Value { return fw.sym.MemVal() }

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
		return core.Nil{}, nil
	}

	if fw.returnsErr {
		errIndex := fw.lastOutIdx + 1
		if !vals[errIndex].IsNil() {
			return nil, vals[errIndex].Interface().(error)
		}

		if fw.rt.NumOut() == 1 {
			return core.Nil{}, nil
		}
	}

	var err error
	retValCount := len(vals[0 : fw.lastOutIdx+1])
	wrapped := make([]ww.Any, retValCount, retValCount)
	for i := 0; i < retValCount; i++ {
		if wrapped[i], err = adaptValue(fw.adapters[i], vals[i]); err != nil {
			return nil, err
		}
	}

	if retValCount == 1 {
		return wrapped[0], nil
	}

	return core.NewList(capnp.SingleSegment(nil), wrapped...)
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

type adapter func(reflect.Value) (ww.Any, error)

func adaptValue(adapt adapter, v reflect.Value) (ww.Any, error) {
	if adapt != nil {
		return adapt(v)
	}

	if any, ok := v.Interface().(ww.Any); ok {
		return any, nil
	}

	return nil, fmt.Errorf("%s is not ww.Any", v.Type())
}

func adapterFor(t reflect.Type) adapter {
	switch t.Kind() {
	case reflect.Bool:
		return toBool

	case reflect.Int, reflect.Uint:
		return toInt

	case reflect.Float32, reflect.Float64:
		return toFloat

	case reflect.String:
		return toString

	case reflect.Slice:
		return maybeNil(toVector)

	case reflect.Array:
		return toVector

	case reflect.Ptr:
		return adapterFor(t.Elem())
	}

	return nil
}

func toBool(v reflect.Value) (ww.Any, error) {
	if v.Bool() {
		return core.True, nil
	}

	return core.False, nil
}

func toInt(v reflect.Value) (ww.Any, error) {
	return core.NewInt64(capnp.SingleSegment(nil), v.Int())
}

func toFloat(v reflect.Value) (ww.Any, error) {
	return core.NewFloat64(capnp.SingleSegment(nil), v.Float())
}

func toString(v reflect.Value) (ww.Any, error) {
	return core.NewString(capnp.SingleSegment(nil), v.String())
}

func toSymbol(v reflect.Value) (ww.Any, error) {
	return core.NewSymbol(capnp.SingleSegment(nil), v.String())
}

func toKeyword(v reflect.Value) (ww.Any, error) {
	return core.NewKeyword(capnp.SingleSegment(nil), v.String())
}

func toVector(v reflect.Value) (ww.Any, error) {
	as := make([]ww.Any, v.Len())
	for i := range as {
		if any, ok := v.Index(i).Interface().(ww.Any); ok {
			as[i] = any
			continue
		}

		return nil, fmt.Errorf("%s is not ww.Any", v.Type())
	}

	return core.NewVector(capnp.SingleSegment(nil), as...)
}

func maybeNil(adapt adapter) adapter {
	return func(v reflect.Value) (ww.Any, error) {
		if v.IsNil() {
			return core.Nil{}, nil
		}

		return adapt(v)
	}
}
