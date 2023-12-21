package exec

import (
	"context"
	"io"
	osexec "os/exec"
)

type Command interface {
	// Wait for command to terminate. It must have been previously started.
	Wait() error

	// StdinPipe returns a pipe that will be connected to the command's standard input when the command starts.
	Stdout() io.ReadCloser

	// StderrPipe returns a pipe that will be connected to the command's standard error when the command starts.
	Stderr() io.ReadCloser
}

type command struct {
	err    chan error
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (c *command) Wait() error {
	err, ok := <-c.err
	// If there's a value in the channel, it means that the command finished with an error
	if ok {
		return err
	}
	return nil
}

func (c *command) Stdout() io.ReadCloser {
	return c.stdout
}

func (c *command) Stderr() io.ReadCloser {
	return c.stderr
}

type Executor struct {
	interpreter interpreter
	dir         string
}

func NewCommandExecutor(dir string) (*Executor, error) {
	interpreter, err := findInterpreter()
	if err != nil {
		return nil, err
	}
	return &Executor{
		interpreter: interpreter,
		dir:         dir,
	}, nil
}

func (e *Executor) StartCommand(ctx context.Context, command string) (Command, error) {
	ec, err := e.interpreter.prepare(command)
	if err != nil {
		return nil, err
	}
	return e.start(ctx, ec)
}

func (e *Executor) start(ctx context.Context, ec *execContext) (Command, error) {
	cmd := osexec.CommandContext(ctx, ec.executable, ec.args...)
	cmd.Dir = e.dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	errCh := make(chan error)

	go func(cmd *osexec.Cmd, errCh chan error) {
		err := cmd.Wait()
		e.interpreter.cleanup(ec)
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}(cmd, errCh)

	return &command{errCh, stdout, stderr}, err
}

func (e *Executor) Exec(ctx context.Context, command string) ([]byte, error) {
	cmd, err := e.StartCommand(ctx, command)
	if err != nil {
		return nil, err
	}

	res, err := io.ReadAll(io.MultiReader(cmd.Stdout(), cmd.Stderr()))
	if err != nil {
		return nil, err
	}

	return res, cmd.Wait()
}
