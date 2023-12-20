package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
)

type Executor struct {
	interpreter interpreter
	dir         string
	scriptFiles []string
}

func NewCommandExecutor(dir string) (*Executor, error) {
	interpreter, err := findInterpreter()
	if err != nil {
		return nil, err
	}
	return &Executor{
		interpreter: interpreter,
		dir:         dir,
		scriptFiles: nil,
	}, nil
}

func (e *Executor) StartCommand(ctx context.Context, command string) (func() error, io.Reader, error) {
	e.interpreter.prepare(command)
	return e.start(ctx)
}

func (e *Executor) start(ctx context.Context) (func() error, io.Reader, error) {
	e.scriptFiles = append(e.scriptFiles, e.interpreter.getScriptFile())
	cmd := osexec.CommandContext(ctx, e.interpreter.getExecutable(), e.interpreter.getArgs()...)
	cmd.Dir = e.dir

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	return cmd.Wait, io.MultiReader(outPipe, errPipe), cmd.Start()
}

func (e *Executor) Exec(ctx context.Context, command string) ([]byte, error) {
	wait, out, err := e.StartCommand(ctx, command)
	if err != nil {
		return nil, err
	}

	res, err := io.ReadAll(out)
	if err != nil {
		return nil, err
	}

	defer e.Cleanup()
	return res, wait()
}

func (e *Executor) Cleanup() {
	if e.scriptFiles != nil {
		for _, file := range e.scriptFiles {
			os.Remove(file)
		}
	}
	e.scriptFiles = nil
}

func findInterpreter() (interpreter, error) {
	interpreter, err := findBashExecutable()
	if err != nil {
		return nil, err
	}

	if interpreter != nil {
		return interpreter, nil
	}

	interpreter, err = findCmdExecutable()
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
	getScriptFile() string
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

func (b *bashInterpreter) getScriptFile() string {
	return b.scriptFile
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

func (c *cmdInterpreter) getScriptFile() string {
	return c.scriptFile
}

func findBashExecutable() (interpreter, error) {
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

func findCmdExecutable() (interpreter, error) {
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
