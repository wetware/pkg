package discover

import (
	"sync"

	log "github.com/lthibault/log/pkg"
)

// Param contains options for discovery queries.  Options passed to DiscoverPeers
// first populate a Param struct.  Fields are exported for the sake of 3rd-party
// discovery implementations.
type Param struct {
	init sync.Once
	log  log.Logger

	Limit int

	// Custom provides a place for 3rd-party Strategies to set implementation-specific
	// options.  As with context.context, developers SHOULD use unexported types as keys
	// to avoid collisions.
	//
	// Note that Custom may be nil.
	Custom map[interface{}]interface{}
}

// Apply options.
func (p *Param) Apply(opt []Option) (err error) {
	for _, f := range opt {
		if err = f(p); err != nil {
			break
		}
	}

	return
}

// Log an event
func (p *Param) Log() log.Logger {
	p.init.Do(func() {
		if p.log == nil {
			p.log = log.New(log.OptLevel(log.NullLevel))
		}
	})

	return p.log
}

// Option modifies the behavior of DiscoverPeers.  Note that the behavior depends on
// the implementation of DiscoverPeers.  Certain options may even be ignored.
type Option func(*Param) error

// WithLogger sets a logger for the duration of the DiscoverPeers call.
func WithLogger(log log.Logger) Option {
	return func(p *Param) error {
		p.log = log
		return nil
	}
}

// WithLimit caps the number of records that can be returned.
func WithLimit(lim int) Option {
	return func(p *Param) error {
		p.Limit = lim
		return nil
	}
}
