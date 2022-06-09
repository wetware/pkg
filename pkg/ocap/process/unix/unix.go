package unix

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	api "github.com/wetware/ww/internal/api/proc"
)

type Executor api.Unix

func (ex Executor) AddRef() Executor {
	return Executor{
		Client: ex.Client.AddRef(),
	}
}

func (ex Executor) Release() {
	ex.Client.Release()
}

// Exec constructs a command and executes it in a native OS process.
func (ex Executor) Exec(ctx context.Context, c CommandFunc) (Proc, capnp.ReleaseFunc) {
	f, release := api.Unix(ex).Exec(ctx, c)
	return Proc(f.Proc()), release
}

type FutureProc api.Executor_exec_Results_Future

// Server implements the Unix Executor capability.  It executes
// a Command in a native OS process.
type Server struct {
	*server.Policy
}

func (s *Server) Client() *capnp.Client {
	return api.Executor_ServerToClient(s, s.Policy).Client
}

// Executor returns an Executor client-capability.  The underlying
// client is constructed with a call to s.Client().
func (s *Server) Executor() Executor {
	return Executor{Client: s.Client()}
}

// Exec constructs a command and executes it in a native OS process.
func (s *Server) Exec(_ context.Context, call api.Executor_exec) error {

	cmd, err := s.bind(call.Args())
	if err != nil {
		return err
	}

	// Abort early if we're unable to allocate results. We don't want to
	// end up with a process we can't control.
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	proc := api.Unix_Proc_ServerToClient(cmd, s.Policy)
	return res.SetProc(api.P(proc))
}

func (s *Server) bind(ps api.Executor_exec_Params) (*cmdServer, error) {
	ptr, err := ps.Param()
	if err != nil {
		return nil, err
	}

	cmd := api.Unix_Command{Struct: ptr.Struct()}
	return newCommandServer(context.Background(), cmd)
}
