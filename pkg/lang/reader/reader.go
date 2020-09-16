package reader

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strings"

	"github.com/spy16/parens"
	"github.com/spy16/parens/reader"
	"github.com/wetware/ww/pkg/lang"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	symbols = map[string]parens.Any{
		"nil":   parens.Nil{},
		"false": lang.False,
		"true":  lang.True,
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

	macroTable = map[rune]macro{
		'"':  {fn: readString},
		';':  {fn: readComment},
		':':  {fn: readKeyword},
		'\\': {fn: readCharacter},
		'(':  {fn: readList},
		')':  {fn: reader.UnmatchedDelimiter()},
		'[':  {fn: readVector},
		']':  {fn: reader.UnmatchedDelimiter()},
		'\'': {fn: quoteFormReader("quote")},
		'~':  {fn: quoteFormReader("unquote")},
		'`':  {fn: quoteFormReader("syntax-quote")},
		'/':  {fn: readPath},
	}

	charLiterals = make(map[string]lang.Char, 6)
)

func init() {
	for k, v := range map[string]rune{
		"tab":       '\t',
		"space":     ' ',
		"newline":   '\n',
		"return":    '\r',
		"backspace": '\b',
		"formfeed":  '\f',
	} {
		c, err := lang.NewChar(capnp.SingleSegment(nil), v)
		if err != nil {
			panic(err)
		}

		charLiterals[k] = c
	}
}

// New returns a lisp reader instance which can read forms from r.
// File name is inferred from the value & type information of 'r' OR
// can be set manually on the Reader instance returned.
func New(r io.Reader) *reader.Reader {
	rd := reader.New(r,
		reader.WithNumReader(readNumber),
		reader.WithPredefinedSymbols(symbols))

	for init, macro := range macroTable {
		rd.SetMacro(init, macro.dispatch, macro.fn)
	}

	return rd
}

func annotateErr(rd *reader.Reader, err error, beginPos reader.Position, form string) error {
	if err == io.EOF || err == reader.ErrSkip {
		return err
	}

	readErr := reader.Error{}
	if e, ok := err.(reader.Error); ok {
		readErr = e
	} else {
		readErr = reader.Error{Cause: err}
	}

	readErr.Form = form
	readErr.Begin = beginPos
	readErr.End = rd.Position()
	return readErr
}

func inferFileName(rs io.Reader) string {
	switch r := rs.(type) {
	case *os.File:
		return r.Name()

	case *strings.Reader:
		return "<string>"

	case *bytes.Reader:
		return "<bytes>"

	case net.Conn:
		return fmt.Sprintf("<conn:%s>", r.LocalAddr())

	default:
		return fmt.Sprintf("<%s>", reflect.TypeOf(rs))
	}
}

type macro struct {
	dispatch bool
	fn       reader.Macro
}
