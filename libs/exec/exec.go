package exec

import (
	"context"
	"io"
	"os"
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
	cmd         *osexec.Cmd
	execContext *execContext
	stdout      io.ReadCloser
	stderr      io.ReadCloser
}

func (c *command) Wait() error {
	// After the command has finished (cmd.Wait call), remove the temporary script file
	defer os.Remove(c.execContext.scriptFile)

	err := c.cmd.Wait()
	if err != nil {
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

	return &command{cmd, ec, stdout, stderr}, cmd.Start()
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
