package builtin

// Option type for analyzer.
type Option func(*Analyzer)

// WithSpecialForms sets the special forms for the analyzer.
func WithSpecialForms(fs map[string]SpecialParser) Option {
	if fs == nil {
		fs = map[string]SpecialParser{
			"do": parseDo,
			"if": parseIf,
			// "fn":    parseFn,
			"def": parseDef,
			// "macro": parseMacro,
			"quote": parseQuote,
			// "go": c.Go,
			"ls": parseLs,
			// "pop":   parsePop,
			// "conj":  parseConj,
		}
	}

	return func(a *Analyzer) {
		a.special = fs
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithSpecialForms(nil),
	}, opt...)
}
