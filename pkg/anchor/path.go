package anchor

import (
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/wetware/ww/pkg/internal/bounded"
)

// Path represents the location of an anchor. It is a bounded value
// that enforces the following constraints:  paths are strings that
// MAY contain any printable ASCII characters except the backslash.
// The forward slash '/' separates path components.
//
// All paths are relative; there is no way of referencing a parent
// anchor from its child. This provides a form of capability-based
// isolation wherein principals are limited to the scope of anchor
// references in their possession.
//
// A path comprising a single '/' is the "root" path. The root path
// designates the current anchor. Equivalently, it can be said that
// the root path is the identity function over anchors, or that the
// Anchor.Walk() method is a nop when the root path is passed in as
// an argument. The empty string automatically resolves to the root
// path. Zero-value paths are also treated as root, but are used as
// as a safe default when the path contains an error, and therefore
// are NOT RECOMMENDED.
type Path struct {
	value bounded.Type[string]
}

// NewPath returns a new Path value, containing a canonical path if
// the 'path' argument is valid, or an error if it is not.
//
// Callers SHOULD check Path.Err() before proceeding.
func NewPath(path string) (p Path) {
	if path = trimmed(path); path != "" {
		p = Path{}.bind(func(s string) bounded.Type[string] {
			return bounded.Value(path)
		})
	}

	return
}

// JoinPath joins each element of 'parts' into a single path,
// with each element separated by '/'.  Each element of 'parts' is
// validated before being joined.   An element is valid if it is a
// valid path, and does not contain the path separator.
//
// Callers SHOULD check Path.Err() before proceeding.
func JoinPath(parts []string) Path {
	return NewPath(path.Join(parts...))
}

// Err returns a non-nil error if the path is malformed.
func (p Path) Err() error {
	_, err := p.value.Maybe()
	return err
}

// String returns the canonical path string.  If Err() != nil,
// String() returns the zero-value string.
func (p Path) String() string {
	s, err := p.value.Maybe()
	if err != nil {
		return ""
	}

	return path.Clean(path.Join("/", s))
}

// Next splits the path into the tail and head components.  It
// is used to iterate through each path component sequentially,
// using the following pattern:
//
//	for path, name := path.Next(); name != ""; path, name = path.Next() {
//	    // do something...
//	}
func (p Path) Next() (Path, string) {
	name := p.bind(head).String()
	return p.bind(tail), trimmed(name)
}

func (p Path) bind(f func(string) bounded.Type[string]) Path {
	value := p.value.
		Bind(f).
		Bind(clean).
		Bind(validate)

	return Path{
		value: value,
	}
}

// Bindable path functions.

func head(path string) bounded.Type[string] {
	path, _ = popleft(path)
	return bounded.Value(path)
}

func tail(path string) bounded.Type[string] {
	_, path = popleft(path)
	return bounded.Value(path)
}

// clean the path through pure lexical analysis, and esure it has
// exactly one leading separator. Cleaned paths are not guaranteed
// to be valid, but are guaranteed to compose within the STM index.
func clean(p string) bounded.Type[string] {
	return bounded.Value(path.Clean(p))
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

// valid returns true if r is a legal character in a path.
// The separator path '/' returns true.
func valid(r rune) bool {
	return unicode.In(r, &pathChars)
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

func pop(index func(string) int, path string) (string, string) {
	if i := index(path); i > 0 {
		return path[:i], path[i:]
	}

	return path, ""
}

func next(path string) int {
	return strings.IndexRune(trimmed(path), '/') + 1
}

func trimmed(path string) string {
	return strings.TrimPrefix(path, "/")
}
