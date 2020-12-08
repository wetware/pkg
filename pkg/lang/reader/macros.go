package reader

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	score "github.com/spy16/slurp/core"
	"github.com/spy16/slurp/reader"
	capnp "zombiezen.com/go/capnproto2"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
)

var symbols = map[string]score.Any{
	"nil":   core.Nil{},
	"false": core.False,
	"true":  core.True,
}

func readSymbol(rd *reader.Reader, init rune) (score.Any, error) {
	beginPos := rd.Position()

	s, err := rd.Token(init)
	if err != nil {
		return nil, annotateErr(rd, err, beginPos, s)
	}

	if predefVal, found := symbols[s]; found {
		return predefVal, nil
	}

	// TODO(performance):  pre-allocate
	return core.NewSymbol(capnp.SingleSegment(nil), s)
}

func readString(rd *reader.Reader, init rune) (score.Any, error) {
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
	return core.NewString(capnp.SingleSegment(nil), b.String())
}

func readComment(rd *reader.Reader, _ rune) (score.Any, error) {
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

func readKeyword(rd *reader.Reader, init rune) (score.Any, error) {
	beginPos := rd.Position()

	token, err := rd.Token(-1)
	if err != nil {
		return nil, annotateErr(rd, err, beginPos, token)
	}

	// TODO(performance):  pre-allocate the arena based on the token length +
	// header length.
	return core.NewKeyword(capnp.SingleSegment(nil), token)
}

func readCharacter(rd *reader.Reader, _ rune) (score.Any, error) {
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
		return core.NewChar(capnp.SingleSegment(nil), runes[0])
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

func readList(rd *reader.Reader, _ rune) (score.Any, error) {
	const listEnd = ')'

	beginPos := rd.Position()

	forms := make([]ww.Any, 0, 32) // pre-allocate to improve performance on small lists
	if err := rd.Container(listEnd, "list", func(val score.Any) error {
		forms = append(forms, val.(ww.Any))
		return nil
	}); err != nil {
		return nil, annotateErr(rd, err, beginPos, "")
	}

	// TODO(performance):  can we pre-allocate here?
	return core.NewList(capnp.SingleSegment(nil), forms...)
}

func readVector(rd *reader.Reader, _ rune) (score.Any, error) {
	const vecEnd = ']'

	beginPos := rd.Position()

	var vec core.Container = core.EmptyVector
	if err := rd.Container(vecEnd, "vector", func(val score.Any) (err error) {
		vec, err = vec.Conj(val.(ww.Any))
		return
	}); err != nil {
		return nil, annotateErr(rd, err, beginPos, "")
	}

	return vec, nil
}

func quoteFormReader(expandFunc string) reader.Macro {
	sym, err := core.NewSymbol(capnp.SingleSegment(nil), expandFunc)
	if err != nil {
		panic(err)
	}

	return func(rd *reader.Reader, _ rune) (score.Any, error) {
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

		return core.NewList(capnp.SingleSegment(nil), sym, expr.(ww.Any))
	}
}

func readPath(rd *reader.Reader, char rune) (_ score.Any, err error) {
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
	return core.NewPath(capnp.SingleSegment(nil), b.String())
}

func readUnicodeChar(token string, base int) (ww.Any, error) {
	num, err := strconv.ParseInt(token, base, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid unicode character: '\\%s'", token)
	}

	if num < 0 || num >= unicode.MaxRune {
		return nil, fmt.Errorf("invalid unicode character: '\\%s'", token)
	}

	// TODO(performance):  pre-allocate arena
	return core.NewChar(capnp.SingleSegment(nil), rune(num))
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
