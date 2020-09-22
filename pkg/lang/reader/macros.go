package reader

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/spy16/parens"
	"github.com/spy16/parens/reader"
	capnp "zombiezen.com/go/capnproto2"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang"
)

var symbols = map[string]parens.Any{
	"nil":   parens.Nil{},
	"false": lang.False,
	"true":  lang.True,
}

func readSymbol(rd *reader.Reader, init rune) (parens.Any, error) {
	beginPos := rd.Position()

	s, err := rd.Token(init)
	if err != nil {
		return nil, annotateErr(rd, err, beginPos, s)
	}

	if predefVal, found := symbols[s]; found {
		return predefVal, nil
	}

	// TODO(performance):  pre-allocate
	return lang.NewSymbol(capnp.SingleSegment(nil), s)
}

func readString(rd *reader.Reader, init rune) (parens.Any, error) {
	beginPos := rd.Position()

	var b strings.Builder
	for {
		r, err := rd.NextRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = reader.ErrEOF
			}
			return nil, annotateErr(rd, err, beginPos, string(init)+b.String())
		}

		if r == '\\' {
			r2, err := rd.NextRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					err = reader.ErrEOF
				}

				return nil, annotateErr(rd, err, beginPos, string(init)+b.String())
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

	// TODO(performance):  pre-allocate the arena based on the string length +
	// header length.
	return lang.NewString(capnp.SingleSegment(nil), b.String())
}

func readComment(rd *reader.Reader, _ rune) (parens.Any, error) {
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

func readKeyword(rd *reader.Reader, init rune) (parens.Any, error) {
	beginPos := rd.Position()

	token, err := rd.Token(-1)
	if err != nil {
		return nil, annotateErr(rd, err, beginPos, token)
	}

	// TODO(performance):  pre-allocate the arena based on the token length +
	// header length.
	return lang.NewKeyword(capnp.SingleSegment(nil), token)
}

func readCharacter(rd *reader.Reader, _ rune) (parens.Any, error) {
	beginPos := rd.Position()

	r, err := rd.NextRune()
	if err != nil {
		return nil, annotateErr(rd, err, beginPos, "")
	}

	token, err := rd.Token(r)
	if err != nil {
		return nil, err
	}
	runes := []rune(token)

	if len(runes) == 1 {
		// TODO(performance):  pre-allocate the arena based on segment header length + 2.
		// 					   N.B.:  rune = int32 => [2]byte
		return lang.NewChar(capnp.SingleSegment(nil), runes[0])
	}

	chr, found := charLiterals[token]
	if found {
		return chr, nil
	}

	if token[0] == 'u' {
		return readUnicodeChar(token[1:], 16)
	}

	return nil, fmt.Errorf("unsupported character: '\\%s'", token)
}

func readList(rd *reader.Reader, _ rune) (parens.Any, error) {
	const listEnd = ')'

	beginPos := rd.Position()

	forms := make([]ww.Any, 0, 32) // pre-allocate to improve performance on small lists
	if err := rd.Container(listEnd, "list", func(val parens.Any) error {
		forms = append(forms, val.(ww.Any))
		return nil
	}); err != nil {
		return nil, annotateErr(rd, err, beginPos, "")
	}

	// TODO(performance):  can we pre-allocate here?
	return lang.NewList(capnp.SingleSegment(nil), forms...)
}

func readVector(rd *reader.Reader, _ rune) (parens.Any, error) {
	const vecEnd = ']'

	beginPos := rd.Position()

	b, err := lang.NewVectorBuilder(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	if err := rd.Container(vecEnd, "vector", b.Conj); err != nil {
		return nil, annotateErr(rd, err, beginPos, "")
	}

	return b.Vector()
}

func quoteFormReader(expandFunc string) reader.Macro {
	sym, err := lang.NewSymbol(capnp.SingleSegment(nil), expandFunc)
	if err != nil {
		panic(err)
	}

	return func(rd *reader.Reader, _ rune) (parens.Any, error) {
		expr, err := rd.One()
		if err != nil {
			if err == io.EOF {
				return nil, reader.Error{
					Form:  expandFunc,
					Cause: reader.ErrEOF,
				}
			} else if err == reader.ErrSkip {
				return nil, reader.Error{
					Form:  expandFunc,
					Cause: errors.New("cannot quote a no-op form"),
				}
			}
			return nil, err
		}

		return parens.NewList(sym, expr), nil
	}
}

func readPath(rd *reader.Reader, char rune) (_ parens.Any, err error) {
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

	// TODO(performance): pre-allocate the arena
	return lang.NewPath(capnp.SingleSegment(nil), b.String())
}

func readUnicodeChar(token string, base int) (lang.Char, error) {
	num, err := strconv.ParseInt(token, base, 64)
	if err != nil {
		return lang.Char{}, fmt.Errorf("invalid unicode character: '\\%s'", token)
	}

	if num < 0 || num >= unicode.MaxRune {
		return lang.Char{}, fmt.Errorf("invalid unicode character: '\\%s'", token)
	}

	// TODO(performance):  pre-allocate arena
	return lang.NewChar(capnp.SingleSegment(nil), rune(num))
}

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
