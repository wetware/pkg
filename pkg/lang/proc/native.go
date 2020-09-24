package proc

// func init() { Register(":go", parseGoroutine) }

// func parseGoroutine(args parens.Seq) (ww.ProcSpec, error) {
// 	target, err := args.First()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return goroutineSpec{target.(ww.Any)}, nil
// }

// type goroutineSpec struct{ ww.Any }

// func (spec goroutineSpec) Start(env *parens.Env) (ww.Proc, error) {
// 	var g goroutine

// 	// We need to ensure the new process is successfully allocated before spawning the
// 	// goroutine.
// 	proc, err := New(capnp.SingleSegment(nil), api.Proc_ServerToClient(&g))
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Now that allocation happened successfully, start the goroutine and assign the
// 	// process to g.
// 	g.Process = goprocess.Go(func(p goprocess.Process) {
// 		// TODO(enhancement):  process should be able to recover this value.
// 		_, _ = env.Eval(spec.Any)
// 	})

// 	return proc, nil
// }

// func (spec goroutineSpec) Params(p api.Anchor_go_Params) error {
// 	pspec, err := p.NewSpec()
// 	if err != nil {
// 		return err
// 	}

// 	gspec, err := pspec.NewGoroutine()
// 	if err != nil {
// 		return err
// 	}

// 	return gspec.SetTarget(spec.MemVal().Raw)
// }

// type goroutine struct{ goprocess.Process }

// func (g goroutine) Wait(call api.Proc_wait) error {
// 	select {
// 	case <-call.Ctx.Done():
// 		return call.Ctx.Err()
// 	case <-g.Closed():
// 		return g.Err()
// 	}
// }
