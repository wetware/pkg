package host

// func subloop(ctx context.Context, env env, f ww.Filter) fx.Hook {
// 	sub := subscriber{
// 		env:    env,
// 		Filter: f,
// 	}

// 	return fx.Hook{
// 		OnStart: func(context.Context) error { return sub.loop(ctx) },
// 		OnStop:  func(context.Context) error { return sub.Close() },
// 	}
// }

// type subscriber struct {
// 	env
// 	ww.Filter

// 	sub     iface.PubSubSubscription
// 	emitter event.Emitter
// 	cancel  context.CancelFunc
// }

// func (s *subscriber) loop(ctx context.Context) (err error) {
// 	ctx, s.cancel = context.WithCancel(ctx)

// 	// TODO:  can we add a custom validator?
// 	s.sub, err = s.Pubsub.Subscribe(ctx, s.Namespace,
// 		options.PubSub.Discover(true))

// 	go s.handleHeartbeat(ctx)
// 	return nil
// }

// func (s subscriber) Close() error {
// 	s.cancel()
// 	return s.sub.Close()
// }

// func (s *subscriber) handleHeartbeat(ctx context.Context) {
// 	defer s.emitter.Close()

// 	for {
// 		switch msg, err := s.sub.Next(ctx); err {
// 		case nil:
// 			hb, err := ww.UnmarshalHeartbeat(msg.Data())
// 			if err != nil {
// 				s.Log.WithError(err).Debug("malformed heartbeat")
// 				continue
// 			}

// 			if !s.Upsert(msg.From(), binary.BigEndian.Uint64(msg.Seq()), hb.TTL()) {
// 				continue
// 			}

// 			event, err := hb.ToEvent()
// 			if err != nil {
// 				s.Log.WithError(err).Debug("heartbeat conversion to event failed")
// 				continue
// 			}

// 			if err = s.emitter.Emit(event); err != nil {
// 				panic(err) // Emit doesn't error unless closed
// 			}
// 		case context.Canceled:
// 			return
// 		default:
// 			s.Log.WithError(err).Debug("error receiving heartbeat")
// 		}

// 	}
// }
