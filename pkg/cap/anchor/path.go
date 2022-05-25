package anchor

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/wetware/ww/pkg/internal/bounded"
)

var root = Path{}.bind(identity)

// PathProvider represents any type capable of returning a path.
// See NewPathFromProvider.
type PathProvider interface {
	// Path returns a path string. The returned string need not be
	// in canonical form, and need not be valid.  Callers MUST NOT
	// assume the result to be well formed, even when the error is
	// nil.
	//
	// The result SHOULD be passed to NewPath promptly.
	Path() (string, error)
}

// PathSetter represents a type capable of receiving a well-formed
// path. Callers MUST pass well-formed paths, though these need not
// be in canonical form.
type PathSetter interface {
	// SetPath assigns the supplied path to the underlying value of
	// PathSetter.  Callers MUST supply a valid paths, but MAY pass
	// non-canonical path strings. In particular, callers MAY strip
	// the leading separator from a path before calling SetPath.
	SetPath(string) error
}

// Path represents the location of an anchor. It is a bounded value
// that enforces the following constraints:  paths are strings that
// MAY contain any printable ASCII characters except the backslash.
// The forward slash '/' separates path components.
//
// All paths are relative; there is no way of referencing a parent
// anchor from its child. This provides a form of capability-based
// isolation wherein principals are limited to the scope of anchor
// references in their posession.
//
// A path comprising a single '/' is the "root" path. The root path
// designates the current anchor. Equivalently, it can be said that
// the root path is the identity function over anchors, or that the
// Anchor.Walk() method is a nop when the root path is passed in as
// an argument. The empty string automatically resolves to the root
// path.
type Path struct {
	value bounded.Type[string]
}

// NewPath returns a new Path value, containing a canonical path if
// the 'path' argument is valid, or an error if it is not.
//
// Callers SHOULD check Path.Err() before proceeding.
func NewPath(path string) Path {
	if trimmed(path) == "" {
		return root
	}

	value := bounded.Value(path)
	return Path{value: value}.bind(identity) // force validation
}

// PathFromParts joins each element of 'parts' into a single path,
// with each element separated by '/'.  Each element of 'parts' is
// validated before being joined.   An element is valid if it is a
// valid path, and does not contain the path separator.
//
// Callers SHOULD check Path.Err() before proceeding.
func PathFromParts(parts []string) Path {
	if err := validateParts(parts); err != nil {
		return failure(err)
	}

	return NewPath(path.Join(parts...))
}

// PathFromProvider constructs a new path from any type that provides
// a Path() (string, error) method.  The resulting path is valid if p
// returns a string without error, and if the string produces a valid
// path when passed to NewPath.
//
// Callers SHOULD check Path.Err() before proceeding.
func PathFromProvider(p PathProvider) Path {
	raw, err := p.Path()
	if err == nil {
		return NewPath(raw)
	}

	// The caller is expected to check the error, and
	// handle it differently from validation errors.
	//
	// However, we don't want to provide users with a way
	// to create valid empty-string Paths, so we return a
	// failed path as well.
	return failure(err)
}

// Err returns a non-nil error if the path is malformed.
func (p Path) Err() error {
	_, err := p.value.Maybe()
	return err
}

// String returns the canonical path string.  If Err() != nil,
// String() returns the root path.
func (p Path) String() (path string) {
	p.bind(func(s string) bounded.Type[string] {
		path = s
		return bounded.Value(s)
	})
	return
}

// IsRoot returns true if the p is the root path.
func (p Path) IsRoot() bool {
	return p.String() == "/"
}

// IsZero reports whether p is a zero-value path, as distinct from
// the root path. If p.IsZero() == true, then p.IsRoot() == false.
// The converse may not be true.
func (p Path) IsZero() bool {
	s, err := p.value.Maybe()
	return s == "" && err == nil
}

// IsChild returns true if path is a direct child of p.
// See also:  p.IsSubpath()
func (p Path) IsChild(path Path) bool {
	parent := p.String()
	candidate := path.String()
	dir, _ := popright(candidate)

	return (parent == dir) != (candidate == "/")
}

// Child binds the child's name to path.  It fails if the
// child name contains invalid characters of a separator.
func (p Path) WithChild(name string) Path {
	if validName(name) {
		return p.bind(suffix(name))
	}

	return failure(errors.New("invalid name"))
}

// Returns true if path is a subpath of p.
func (p Path) IsSubpath(path Path) bool {
	parent := p.String()
	candidate := path.String()
	diff := strings.TrimPrefix(candidate, parent)

	return diff != "" && (parent == "/" || diff[0] == '/')
}

func (p Path) Next() (Path, string) {
	name := p.bind(head).String()
	return p.bind(tail), trimmed(name)
}

// Param binds the path to the supplied PathSetter, stripping
// the leading separator.  If successful, it returns the path
// unmodified.
func (p Path) Bind(target PathSetter) error {
	return p.bind(func(path string) bounded.Type[string] {
		err := target.SetPath(trimmed(path))
		return bounded.Failure[string](err) // can be nil
	}).Err()
}

func (p Path) index() []byte {
	path := p.String()
	return []byte(path) // TODO(performance):  unsafe.Pointer
}

func (p Path) bind(f func(string) bounded.Type[string]) Path {
	value := p.value.
		Bind(clean).
		Bind(validate).
		Bind(f)

	return Path{
		value: value,
	}
}

// Bindable path functions.

func subpath(path Path) func(string) bounded.Type[string] {
	return suffix(path.String())
}

func trimPrefix(path Path) func(string) bounded.Type[string] {
	return func(s string) bounded.Type[string] {
		suffix := strings.TrimPrefix(path.String(), s)
		return bounded.Value(suffix).Bind(clean)
	}
}

func identity(path string) bounded.Type[string] {
	return bounded.Value(path)
}

func head(path string) bounded.Type[string] {
	path, _ = popleft(path)
	return bounded.Value(path)
}

func tail(path string) bounded.Type[string] {
	_, path = popleft(path)
	return bounded.Value(path)
}

func last(path string) bounded.Type[string] {
	_, path = popright(path)
	return bounded.Value(path)
}

func suffix(s string) func(string) bounded.Type[string] {
	return func(prefix string) bounded.Type[string] {
		return bounded.Value(path.Join(prefix, s))
	}
}

// clean the path through pure lexical analysis, and esure it has
// exactly one leading separator. Cleaned paths are not guaranteed
// to be valid, but are guaranteed to compose within the STM index.
func clean(p string) bounded.Type[string] {
	// Ensure the path begins with a "/", so that prefixes
	// compose well in the STM index.
	p = path.Clean(path.Join("/", p))
	return bounded.Value(p)
}

// validate returns the unmodified path if it contains only
// valid characters.
func validate(path string) bounded.Type[string] {
	for i, r := range path {
		if !valid(r) {
			err := fmt.Errorf("invalid rune '%c' at index %d", i, r)
			return bounded.Failure[string](err)
		}

	}

	return bounded.Value(path)
}

func validateParts(path []string) error {
	// ensure there are no path separators in the components.
	for i, p := range path {
		if !validName(p) {
			return fmt.Errorf("segment %d: invalid name", i)
		}
	}

	return nil
}

// valid returns true if r is a legal character in a path.
// The separator path '/' returns true.
func valid(r rune) bool {
	return unicode.In(r, &pathChars)
}

func validName(name string) bool {
	for _, r := range name {
		if r == '/' || !valid(r) {
			return false
		}
	}

	return true
}

func failure(err error) Path {
	return Path{
		value: bounded.Failure[string](err),
	}
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

func popleft(path string) (string, string) {
	return pop(next, path)
}

func popright(path string) (string, string) {
	return pop(prev, path)
}

func pop(index func(string) int, path string) (string, string) {
	if i := index(path); i > 0 {
		return path[:i], path[i:]
	}

	return path, ""
}

func next(path string) int {
	return strings.IndexRune(trimmed(path), '/') + 1
}

func prev(path string) int {
	i := strings.LastIndex(trimmed(path), "/")
	if i < 0 {
		i = 0
	}

	return i + 1
}

func trimmed(path string) string {
	return strings.TrimPrefix(path, "/")
}
