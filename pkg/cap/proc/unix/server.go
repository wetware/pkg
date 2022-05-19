package unix

import (
	"context"
	"io"
	"os/exec"

	api "github.com/wetware/ww/internal/api/proc"
)

type Server struct{}

func (s *Server) NewClient() *Client {
	return &Client{client: api.Executor_ServerToClient(s, nil)}
}

func (s *Server) Exec(ctx context.Context, call api.Executor_exec) error {
	ptr, err := call.Args().Profile()
	if err != nil {
		return err
	}

	unixCmd := api.UnixCommand{Struct: ptr.Struct()}

	name, err := unixCmd.Name()
	if err != nil {
		return err
	}

	arg, err := unixCmd.Arg()
	if err != nil {
		return err
	}

	args := make([]string, 0, arg.Len())

	for i := 0; i < arg.Len(); i++ {
		a, err := arg.At(i)
		if err != nil {
			return err
		}
		args = append(args, a)
	}

	cmd := exec.Command(name, args...)
	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	return results.SetProc(api.Process_ServerToClient(&ProcessServer{cmd: cmd}, nil))
}

type ProcessServer struct {
	cmd *exec.Cmd
}

func (s *ProcessServer) Start(context.Context, api.Process_start) error {
	return s.cmd.Start()
}

func (s *ProcessServer) Wait(context.Context, api.Process_wait) error {
	return s.cmd.Wait()
}

func (s *ProcessServer) StderrPipe(ctx context.Context, call api.Process_stderrPipe) error {
	rc, err := s.cmd.StderrPipe()
	if err != nil {
		return err
	}

	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	return results.SetRc(api.ReadCloser_ServerToClient(&ReadCloserServer{rc: rc}, nil))
}

func (s *ProcessServer) StdinPipe(_ context.Context, call api.Process_stdinPipe) error {
	wc, err := s.cmd.StdinPipe()
	if err != nil {
		return err
	}

	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	return results.SetWc(api.WriteCloser_ServerToClient(&WriteCloserServer{wc: wc}, nil))
}

func (s *ProcessServer) StdoutPipe(ctx context.Context, call api.Process_stdoutPipe) error {
	rc, err := s.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	return results.SetRc(api.ReadCloser_ServerToClient(&ReadCloserServer{rc: rc}, nil))
}

type ReadCloserServer struct {
	rc io.ReadCloser
}

func (s *ReadCloserServer) Read(ctx context.Context, call api.Reader_read) error {
	buffer := make([]byte, call.Args().N())
	n, err := s.rc.Read(buffer)
	if err != nil {
		return err
	}

	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	results.SetN(int64(n))
	return results.SetData(buffer)
}

func (s *ReadCloserServer) Close(context.Context, api.Closer_close) error {
	return s.rc.Close()
}

type WriteCloserServer struct {
	wc io.WriteCloser
}

func (s *WriteCloserServer) Write(ctx context.Context, call api.Writer_write) error {
	b, err := call.Args().Data()
	if err != nil {
		return err
	}

	n, err := s.wc.Write(b)
	if err != nil {
		return err
	}

	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	results.SetN(int64(n))
	return nil
}

func (s *WriteCloserServer) Close(context.Context, api.Closer_close) error {
	return s.wc.Close()
}
