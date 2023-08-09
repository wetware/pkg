package survey

import (
	"github.com/libp2p/go-libp2p/core/discovery"
)

/*
	options.go contains discovery.Option types for surveyors
*/

type (
	keyDistance struct{}
)

func distance(o discovery.Options) uint8 {
	if d, ok := o.Other[keyDistance{}].(uint8); ok {
		return d
	}

	return 255
}

// option for specifying distance when calling FindPeers
func WithDistance(dist uint8) discovery.Option {
	return func(opts *discovery.Options) error {
		opts.Other = make(map[interface{}]interface{})
		opts.Other[keyDistance{}] = dist
		return nil
	}
}
