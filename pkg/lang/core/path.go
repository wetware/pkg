package core

import (
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"github.com/spy16/sabre/reader"
	"github.com/spy16/sabre/runtime"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

var _ runtime.Value = Path(nil)

// Path points to an anchor
type Path []string

func pathFromString(s string) Path {
	return anchorpath.Parts(s)
}

func (p Path) String() string {
	return anchorpath.Join(p)
}

// Eval .
func (p Path) Eval(r runtime.Runtime) (runtime.Value, error) {
	return p, nil
}

// pathMacro implements native path notation in the REPL.
func pathMacro() reader.Macro {
	return func(rd *reader.Reader, char rune) (_ runtime.Value, err error) {
		var b strings.Builder
		for {
			b.WriteRune(char)

			if char, err = rd.NextRune(); err != nil {
				return
			}

			if isPathTerminator(char) {
				rd.Unread(char)
				break
			}

			if !isValidPathChar(char) {
				return nil, runtime.NewErr(
					true,
					rd.Position(),
					errors.Errorf("invalid path character %q", char),
				)

			}
		}

		return pathFromString(b.String()), nil
	}
}

func isPathTerminator(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}

	switch r {
	case ')', ']', '}', ',':
		return true
	default:
		return false
	}
}

func isValidPathChar(r rune) bool {
	switch r {
	case '/', '-', '_', '.', '~':
		return true
	default:
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
}
