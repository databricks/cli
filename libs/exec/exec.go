package exec

import (
	"context"
	"errors"
	"fmt"
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
	done   chan bool
	err    chan error
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (c *command) Wait() error {
	select {
	case <-c.done:
		return nil
	case err := <-c.err:
		return err
	}
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
	e.interpreter.prepare(command)
	return e.start(ctx)
}

func (e *Executor) start(ctx context.Context) (Command, error) {
	cmd := osexec.CommandContext(ctx, e.interpreter.getExecutable(), e.interpreter.getArgs()...)
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

	done := make(chan bool)
	errCh := make(chan error)

	go func() {
		err := cmd.Wait()
		e.interpreter.cleanup()
		if err != nil {
			errCh <- err
		}
		close(done)
		close(errCh)
	}()

	return &command{done, errCh, stdout, stderr}, err
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

func findInterpreter() (interpreter, error) {
	interpreter, err := findBashInterpreter()
	if err != nil {
		return nil, err
	}

	if interpreter != nil {
		return interpreter, nil
	}

	interpreter, err = findCmdInterpreter()
	if err != nil {
		return nil, err
	}

	if interpreter != nil {
		return interpreter, nil
	}

	return nil, errors.New("no interpreter found")
}

type interpreter interface {
	prepare(command string) error
	getExecutable() string
	getArgs() []string
	cleanup() error
}

type bashInterpreter struct {
	executable string
	args       []string
	scriptFile string
}

func (b *bashInterpreter) prepare(command string) error {
	filename, err := createTempScript(command, ".sh")
	if err != nil {
		return err
	}

	b.args = []string{"-e", filename}
	b.scriptFile = filename

	return nil
}

func (b *bashInterpreter) getExecutable() string {
	return b.executable
}

func (b *bashInterpreter) getArgs() []string {
	return b.args
}

func (b *bashInterpreter) cleanup() error {
	return os.Remove(b.scriptFile)
}

type cmdInterpreter struct {
	executable string
	args       []string
	scriptFile string
}

func (c *cmdInterpreter) prepare(command string) error {
	filename, err := createTempScript(command, ".cmd")
	if err != nil {
		return err
	}

	c.args = []string{"/D", "/E:ON", "/V:OFF", "/S", "/C", fmt.Sprintf(`CALL %s`, filename)}
	c.scriptFile = filename

	return nil
}

func (c *cmdInterpreter) getExecutable() string {
	return c.executable
}

func (c *cmdInterpreter) getArgs() []string {
	return c.args
}

func (c *cmdInterpreter) cleanup() error {
	return os.Remove(c.scriptFile)
}

func findBashInterpreter() (interpreter, error) {
	// Lookup for bash executable first (Linux, MacOS, maybe Windows)
	out, err := osexec.LookPath("bash")
	if err != nil && !errors.Is(err, osexec.ErrNotFound) {
		return nil, err
	}

	// Bash executable is not found, returning early
	if out == "" {
		return nil, nil
	}

	return &bashInterpreter{executable: out}, nil
}

func findCmdInterpreter() (interpreter, error) {
	// Lookup for CMD executable (Windows)
	out, err := osexec.LookPath("cmd")
	if err != nil && !errors.Is(err, osexec.ErrNotFound) {
		return nil, err
	}

	// CMD executable is not found, returning early
	if out == "" {
		return nil, nil
	}

	return &cmdInterpreter{executable: out}, nil
}

func createTempScript(command string, extension string) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "cli-exec*"+extension)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = io.WriteString(file, command)
	if err != nil {
		// Try to remove the file if we failed to write to it
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}
