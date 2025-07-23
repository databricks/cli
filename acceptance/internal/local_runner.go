package internal

import (
	"context"
	"io"
	"os"
	"os/exec"
)

type LocalRunner struct {
	ctx        context.Context
	args       []string
	env        []string
	dir        string
	cmd        *exec.Cmd
	output     io.ReadCloser
	writer     io.WriteCloser
	exitCode   int
	cancelFunc func() error
}

func NewLocalRunner(ctx context.Context) *LocalRunner {
	return &LocalRunner{
		ctx: ctx,
	}
}

func (r *LocalRunner) SetArgs(args []string) {
	r.args = args
}

func (r *LocalRunner) SetEnv(env []string) {
	r.env = env
}

func (r *LocalRunner) AddEnv(env string) {
	if r.env == nil {
		r.env = []string{}
	}
	r.env = append(r.env, env)
}

func (r *LocalRunner) SetDir(dir string) {
	r.dir = dir
}

func (r *LocalRunner) Start() error {
	// Create command with stored parameters (not using context to match original behavior)
	r.cmd = exec.Command(r.args[0], r.args[1:]...)
	r.cmd.Dir = r.dir
	r.cmd.Env = r.env

	// Apply the cancel function if it was set
	if r.cancelFunc != nil {
		r.cmd.Cancel = r.cancelFunc
	}

	// Create pipe for output
	reader, writer := io.Pipe()
	r.cmd.Stdout = writer
	r.cmd.Stderr = writer
	r.output = reader
	r.writer = writer

	// Start the command
	err := r.cmd.Start()
	if err != nil {
		writer.Close()
		return err
	}

	return nil
}

func (r *LocalRunner) Run() error {
	err := r.Start()
	if err != nil {
		return err
	}
	return r.Wait()
}

func (r *LocalRunner) Wait() error {
	if r.cmd == nil {
		return nil
	}

	err := r.cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			r.exitCode = exitError.ExitCode()
		} else {
			r.exitCode = 1
		}
	} else {
		r.exitCode = 0
	}

	// Close the writer to signal EOF to readers
	if r.writer != nil {
		r.writer.Close()
	}

	return err
}

func (r *LocalRunner) SetCancelFunc(cancelFunc func() error) {
	r.cancelFunc = cancelFunc
}

func (r *LocalRunner) Kill() error {
	if r.cmd != nil {
		return r.cmd.Process.Kill()
	}
	if r.writer != nil {
		r.writer.Close()
	}
	return nil
}

func (r *LocalRunner) GetExitCode() int {
	return r.exitCode
}

func (r *LocalRunner) Output() io.ReadCloser {
	return r.output
}

func (r *LocalRunner) GetProcess() *os.Process {
	if r.cmd != nil {
		return r.cmd.Process
	}
	return nil
}

func (r *LocalRunner) GetWriter() io.WriteCloser {
	return r.writer
}
