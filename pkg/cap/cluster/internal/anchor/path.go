package anchor

import (
	"fmt"
	"path"
	"strings"
	"unicode"
	"unsafe"

	"github.com/wetware/ww/pkg/internal/bounded"
)

type PathProvider interface {
	Path() (string, error)
}

type PathSetter interface {
	SetPath(string) error
}

type Path bounded.Type[string]

func NewPath(s string) Path {
	path := bounded.Value(s)
	return Path(path).
		Bind(clean).
		Bind(validate)
}

func PathFromParts(parts []string) Path {
	if err := validateParts(parts); err != nil {
		return failure(err)
	}

	return NewPath(path.Join(parts...))
}

func PathFromProvider(p PathProvider) (Path, error) {
	raw, err := p.Path()
	if err == nil {
		return NewPath(raw), nil
	}

	// The caller is expected to check the error, and
	// handle it differently from validation errors.
	//
	// However, we don't want to provide users with a way
	// to create valid empty-string Paths, so we return a
	// failed path as well.
	return failure(err), err
}

func (p Path) Err() error {
	_, err := bounded.Type[string](p).Maybe()
	return err
}

func (p Path) String() string {
	s, _ := bounded.Type[string](p).Maybe()
	return s
}

func (p Path) index() (index []byte) {
	if path := p.String(); path != "/" {
		index = *(*[]byte)(unsafe.Pointer(&path))
	}

	return
}

func (p Path) Next() (Path, string) {
	raw := p.String()

	for i, r := range raw {
		if i > 0 && r == '/' {
			return value(raw[i:]), raw[:i]
		}
	}

	return Path{}, raw
}

func (p Path) Bind(f func(string) bounded.Type[string]) Path {
	value := bounded.Type[string](p)
	value = value.Bind(f)
	return Path(value)
}

func Child(name string) func(string) bounded.Type[string] {
	return func(parent string) bounded.Type[string] {
		if err := validateName(name); err != nil {
			return bounded.Failure[string](err)
		}

		return bounded.Value(parent + "/" + name)
	}
}

func Param(p PathSetter) func(string) bounded.Type[string] {
	return func(s string) bounded.Type[string] {
		s = strings.TrimLeft(s, "/")                 // trim leading '/'
		return bounded.Failure[string](p.SetPath(s)) // error can be nil
	}
}

func clean(p string) bounded.Type[string] {
	// Ensure the path begins with a "/", so that prefixes
	// compose well in the STM index.
	p = path.Clean(path.Join("/", p))
	return bounded.Value(p)
}

func validate(path string) bounded.Type[string] {
	for i, r := range path {
		if valid(r) {
			continue
		}

		err := fmt.Errorf("invalid rune '%c' at index %d", i, r)
		return bounded.Failure[string](err)

	}

	return bounded.Value(path)
}

func valid(r rune) bool {
	return unicode.In(r, &pathChars)
}

func validateParts(path []string) error {
	// ensure there are no path separators in the components.
	for i, p := range path {
		if err := validateName(p); err != nil {
			return fmt.Errorf("path segment %d: %w", i, err)
		}
	}

	return nil
}

func validateName(name string) error {
	for i, r := range name {
		if r == '/' {
			return fmt.Errorf("invalid rune '%c' (index=%d)", r, i)
		}
	}

	return nil
}

// WARNING:  DO NOT use this on unvalidated data.
//           In fact, don't use this where you could
//           be using NewValue.
func value(s string) Path {
	return Path(bounded.Value(s))
}

func failure(err error) Path {
	path := bounded.Failure[string](err)
	return Path(path)
}

// Paths can contain any printable ASCII character, except
// for the backslash '\' (0x005C), which is easily confused
// with '/' (0x002F).
var pathChars = unicode.RangeTable{
	R16: []unicode.Range16{
		{0x0021, 0x005B, 1},
		{0x005D, 0x007E, 1},
	},
	LatinOffset: 2,
}
