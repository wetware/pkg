package lang

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/spy16/sabre/reader"
	"github.com/spy16/sabre/runtime"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	readerMacros = reader.MacroTable{
		'"':  readString,
		';':  readComment,
		':':  readKeyword,
		'\\': readCharacter,
		'(':  readList,
		')':  reader.UnmatchedDelimiter(),
		'[':  readVector,
		']':  reader.UnmatchedDelimiter(),
		// '\'': quoteFormReader("quote"),
		// '~':  quoteFormReader("unquote"),
		// '`':  quoteFormReader("syntax-quote"),
		'/': readPath,
	}

	symbols = map[string]runtime.Value{
		"nil":   core.Nil{},
		"false": core.False,
		"true":  core.True,
	}

	escapeMap = map[rune]rune{
		'"':  '"',
		'n':  '\n',
		'\\': '\\',
		't':  '\t',
		'a':  '\a',
		'f':  '\a',
		'r':  '\r',
		'b':  '\b',
		'v':  '\v',
	}

	charLiterals = map[string]rune{
		"tab":       '\t',
		"space":     ' ',
		"newline":   '\n',
		"return":    '\r',
		"backspace": '\b',
		"formfeed":  '\f',
	}
)

// NewReader .
func NewReader(r io.Reader) *reader.Reader {
	return reader.New(r,
		reader.WithMacros(readerMacros),
		reader.WithPredefinedSymbols(symbols))
}

func readString(rd *reader.Reader, _ rune) (runtime.Value, error) {
	var b strings.Builder

	for {
		r, err := rd.NextRune()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("%w: while reading string", reader.ErrEOF)
			}

			return nil, err
		}

		if r == '\\' {
			r2, err := rd.NextRune()
			if err != nil {
				if err == io.EOF {
					return nil, fmt.Errorf("%w: while reading string", reader.ErrEOF)
				}

				return nil, err
			}

			// TODO: Support for Unicode escape \uNN format.

			escaped, err := getEscape(r2)
			if err != nil {
				return nil, err
			}
			r = escaped
		} else if r == '"' {
			break
		}

		b.WriteRune(r)
	}

	return core.NewString(b.String())
}

func readComment(rd *reader.Reader, _ rune) (runtime.Value, error) {
	for {
		r, err := rd.NextRune()
		if err != nil {
			return nil, err
		}

		if r == '\n' {
			break
		}
	}

	return nil, reader.ErrSkip
}

func readKeyword(rd *reader.Reader, init rune) (runtime.Value, error) {
	token, err := rd.Token(-1)
	if err != nil {
		return nil, err
	}

	return core.NewKeyword(token)
}

func readCharacter(rd *reader.Reader, _ rune) (runtime.Value, error) {
	r, err := rd.NextRune()
	if err != nil {
		return nil, fmt.Errorf("%w: while reading character", reader.ErrEOF)
	}

	token, err := rd.Token(r)
	if err != nil {
		return nil, err
	}
	runes := []rune(token)

	if len(runes) == 1 {
		return core.NewChar(runes[0])
	}

	v, found := charLiterals[token]
	if found {
		return runtime.Char(v), nil
	}

	if token[0] == 'u' {
		return readUnicodeChar(token[1:], 16)
	}

	return nil, fmt.Errorf("unsupported character: '\\%s'", token)
}

func readPath(rd *reader.Reader, char rune) (_ runtime.Value, err error) {
	var b strings.Builder
	for {
		b.WriteRune(char)

		if char, err = rd.NextRune(); err != nil {
			return
		}

		if char != '/' && rd.IsTerminal(char) {
			rd.Unread(char)
			break
		}
	}

	return core.NewPath(b.String())
}

func readList(rd *reader.Reader, _ rune) (runtime.Value, error) {
	const listEnd = ')'

	forms, err := rd.Container(listEnd, "list")
	if err != nil {
		return nil, err
	}
	return runtime.NewSeq(forms...), nil
}

func readVector(rd *reader.Reader, char rune) (runtime.Value, error) {
	b, err := core.NewVectorBuilder(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	if err = readContainerStream(rd, ']', "vector", func(v runtime.Value) error {
		return b.Conj(v)
	}); err != nil {
		return nil, err
	}

	return b.Vector()
}

func readUnicodeChar(token string, base int) (core.Char, error) {
	num, err := strconv.ParseInt(token, base, 64)
	if err != nil {
		return core.Char{}, fmt.Errorf("invalid unicode character: '\\%s'", token)
	}

	if num < 0 || num >= unicode.MaxRune {
		return core.Char{}, fmt.Errorf("invalid unicode character: '\\%s'", token)
	}

	return core.NewChar(rune(num))
}

// func quoteFormReader(expandFunc string) reader.Macro {
// 	return func(rd *reader.Reader, _ rune) (runtime.Value, error) {
// 		expr, err := rd.One()
// 		if err != nil {
// 			if err == io.EOF {
// 				return nil, fmt.Errorf("%w: while reading quote form", reader.ErrEOF)
// 			} else if err == reader.ErrSkip {
// 				return nil, errors.New("no-op form while reading quote form")
// 			}
// 			return nil, err
// 		}

// 		// TODO(xxx):  replace NewSeq with something else; it uses sabre's linked list
// 		//			   under the hood.
// 		return runtime.NewSeq(
// 			runtime.Symbol{Value: expandFunc},
// 			expr,
// 		), nil
// 	}
// }

func getEscape(r rune) (rune, error) {
	escaped, found := escapeMap[r]
	if !found {
		return -1, fmt.Errorf("illegal escape sequence '\\%c'", r)
	}

	return escaped, nil
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r) || r == ','
}

func readContainerStream(rd *reader.Reader, end rune, formType string, fn func(runtime.Value) error) error {
	for {
		if err := rd.SkipSpaces(); err != nil {
			if err == io.EOF {
				return fmt.Errorf("%w: while reading %s", reader.ErrEOF, formType)
			}
			return err
		}

		r, err := rd.NextRune()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("%w: while reading %s", reader.ErrEOF, formType)
			}
			return err
		}

		if r == end {
			break
		}
		rd.Unread(r)

		expr, err := rd.One()
		if err != nil {
			if err == reader.ErrSkip {
				continue
			}
			return err
		}

		if err = fn(expr); err != nil {
			return err
		}
	}

	return nil
}
