package client

type Option func(*Dialer)

func WithHost(h HostFactory) Option {
	if h == nil {
		h = &BasicHostFactory{}
	}

	return func(d *Dialer) {
		d.host = h
	}
}

func WithRouting(r RoutingFactory) Option {
	if r == nil {
		r = defaultRoutingFactory{}
	}

	return func(d *Dialer) {
		d.routing = r
	}
}

func WithPubSub(p PubSubFactory) Option {
	if p == nil {
		p = defaultPubSubFactory{}
	}

	return func(d *Dialer) {
		d.pubsub = p
	}
}

func WithRPCFactory(r RPCFactory) Option {
	if r == nil {
		r = BasicRPCFactory{}
	}

	return func(d *Dialer) {
		d.rpc = r
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithHost(nil),
		WithRouting(nil),
		WithPubSub(nil),
		WithRPCFactory(nil),
	}, opt...)
}
