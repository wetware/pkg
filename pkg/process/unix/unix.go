package unix

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/proc"
)

type Executor api.Unix

func (ex Executor) AddRef() Executor {
	return Executor(capnp.Client(ex).AddRef())
}

func (ex Executor) Release() {
	capnp.Client(ex).Release()
}

// Exec constructs a command and executes it in a native OS process.
func (ex Executor) Exec(ctx context.Context, c CommandFunc) Proc {
	f, release := api.Unix(ex).Exec(ctx, c)
	proc := Proc(f.Proc()) // must happen before 'go' to avoid race
	go func() {
		defer release()
		<-f.Done()
	}()

	return proc
}

type FutureProc api.Executor_exec_Results_Future

// Server implements the Unix Executor capability.  It executes
// a Command in a native OS process.
type Server struct{}

func (s *Server) Client() capnp.Client {
	return capnp.Client(s.Executor())
}

// Executor returns an Executor client-capability.  The underlying
// client is constructed with a call to s.Client().
func (s *Server) Executor() Executor {
	return Executor(api.Executor_ServerToClient(s))
}

// Exec constructs a command and executes it in a native OS process.
func (s *Server) Exec(_ context.Context, call api.Executor_exec) error {
	// Abort early if we're unable to allocate results. We don't want to
	// end up with a process we can't control.
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	var h handle
	if err = s.bind(&h, call.Args()); err == nil {
		proc := api.Unix_Proc_ServerToClient(&h)
		err = res.SetProc(api.Waiter(proc))
	}

	return err
}

func (s *Server) bind(h *handle, ps api.Executor_exec_Params) error {
	ptr, err := ps.Param()
	if err == nil {
		cmd := api.Unix_Command(ptr.Struct())
		return h.bind(context.Background(), cmd)
	}

	return err
}
