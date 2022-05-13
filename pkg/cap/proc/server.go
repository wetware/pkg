package proc

import (
	"context"
	"io"
	"os/exec"

	api "github.com/wetware/ww/internal/api/proc"
)

type Server struct{}

func (s *Server) NewClient() *Client {
	return &Client{client: api.UnixExecutor_ServerToClient(s, nil)}
}

func (s *Server) Command(ctx context.Context, call api.UnixExecutor_command) error {
	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	arg, err := call.Args().Arg()
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

	return results.SetCmd(api.Cmd_ServerToClient(&CmdServer{cmd: cmd}, nil))
}

type CmdServer struct {
	cmd *exec.Cmd
}

func (s *CmdServer) Start(context.Context, api.Cmd_start) error {
	return s.cmd.Start()
}

func (s *CmdServer) Wait(context.Context, api.Cmd_wait) error {
	return s.cmd.Wait()
}

func (s *CmdServer) StderrPipe(ctx context.Context, call api.Cmd_stderrPipe) error {
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

func (s *CmdServer) StdinPipe(_ context.Context, call api.Cmd_stdinPipe) error {
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

func (s *CmdServer) StdoutPipe(ctx context.Context, call api.Cmd_stdoutPipe) error {
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
	return results.SetP(buffer)
}

func (s *ReadCloserServer) Close(context.Context, api.Closer_close) error {
	return s.rc.Close()
}

type WriteCloserServer struct {
	wc io.WriteCloser
}

func (s *WriteCloserServer) Write(ctx context.Context, call api.Writer_write) error {
	b, err := call.Args().P()
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
