package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
)

type ExecutableType string

const (
	BashExecutable ExecutableType = `bash`
	ShExecutable   ExecutableType = `sh`
	CmdExecutable  ExecutableType = `cmd`
)

var finders map[ExecutableType](func() (shell, error)) = map[ExecutableType](func() (shell, error)){
	BashExecutable: newBashShell,
	ShExecutable:   newShShell,
	CmdExecutable:  newCmdShell,
}

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
	defer c.execContext.cleanup()

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
	shell         shell
	dir           string
	inheritOutput bool
}

// NewCommandExecutor creates an Executor with default output behavior (no inheritance)
func NewCommandExecutor(dir string) (*Executor, error) {
	shell, err := findShell()
	if err != nil {
		return nil, err
	}
	return &Executor{
		shell:         shell,
		dir:           dir,
		inheritOutput: false,
	}, nil
}

func (e *Executor) WithInheritOutput() {
	e.inheritOutput = true
}

func NewCommandExecutorWithExecutable(dir string, execType ExecutableType) (*Executor, error) {
	f, ok := finders[execType]
	if !ok {
		return nil, fmt.Errorf("%s is not supported as an artifact executable, options are: %s, %s or %s", execType, BashExecutable, ShExecutable, CmdExecutable)
	}
	shell, err := f()
	if err != nil {
		return nil, err
	}

	return &Executor{
		shell: shell,
		dir:   dir,
	}, nil
}

func (e *Executor) prepareCommand(ctx context.Context, command string) (*osexec.Cmd, *execContext, error) {
	ec, err := e.shell.prepare(command)
	if err != nil {
		return nil, nil, err
	}
	cmd := osexec.CommandContext(ctx, ec.executable, ec.args...)
	cmd.Dir = e.dir
	return cmd, ec, nil
}

func (e *Executor) StartCommand(ctx context.Context, command string) (Command, error) {
	cmd, ec, err := e.prepareCommand(ctx, command)
	if err != nil {
		return nil, err
	}
	return e.start(cmd, ec)
}

func (e *Executor) start(cmd *osexec.Cmd, ec *execContext) (Command, error) {
	if e.inheritOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return &command{cmd, ec, nil, nil}, cmd.Start()
	}
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
	cmd, ec, err := e.prepareCommand(ctx, command)
	if err != nil {
		return nil, err
	}
	defer ec.cleanup()
	return cmd.CombinedOutput()
}

func (e *Executor) ShellType() ExecutableType {
	return e.shell.getType()
}
