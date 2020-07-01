package boot

// Param contains options for discovery queries.  Options passed to DiscoverPeers
// first populate a Param struct.  Fields are exported for the sake of 3rd-party
// discovery implementations.
type Param struct {
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

// Option modifies the behavior of DiscoverPeers.  Note that the behavior depends on
// the implementation of DiscoverPeers.  Certain options may even be ignored.
type Option func(*Param) error

// WithLimit caps the number of records that can be returned.
func WithLimit(lim int) Option {
	return func(p *Param) error {
		p.Limit = lim
		return nil
	}
}
