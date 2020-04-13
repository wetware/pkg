package repoutil

import (
	"io"
	"io/ioutil"

	config "github.com/ipfs/go-ipfs-config"
)

// Option for repo creation
type Option func(*initSpec)

// WithPrinter specifies a printer, which will log information about repository
// initialization.
func WithPrinter(p io.Writer) Option {
	return func(s *initSpec) {
		s.Printer = p
	}
}

// WithKeySize specifies the keysize for the repository's cryptographic keys.
// Must be power of 2.
func WithKeySize(size int) Option {
	return func(s *initSpec) {
		s.KeySize = size
	}
}

// WithConfig specifies the IPFS config that should be written to the new repository.
func WithConfig(cfg *config.Config) Option {
	return func(s *initSpec) {
		s.Config = cfg
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithPrinter(ioutil.Discard),
		WithKeySize(DefaultKeySize),
	}, opt...)
}

type initSpec struct {
	Printer io.Writer
	KeySize int
	*config.Config
}

func specWithOptions(opt []Option) *initSpec {
	var spec initSpec
	spec.apply(opt)
	return &spec
}

func (s *initSpec) apply(opt []Option) {
	for _, f := range withDefault(opt) {
		f(s)
	}
}
