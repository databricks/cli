package cmdio

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

type Executor struct {
	dir string
}

func NewCommandExecutor(dir string) *Executor {
	return &Executor{
		dir: dir,
	}
}

func (e *Executor) StartCommand(ctx context.Context, command string) (*exec.Cmd, io.Reader, error) {
	interpreter, err := findInterpreter()
	if err != nil {
		return nil, nil, err
	}

	args := interpreter.args
	args = append(args, command)
	cmd := exec.CommandContext(ctx, interpreter.executable, args...)
	cmd.Dir = e.dir

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	return cmd, io.MultiReader(outPipe, errPipe), cmd.Start()

}

func (e *Executor) Exec(ctx context.Context, command string) ([]byte, error) {
	cmd, out, err := e.StartCommand(ctx, command)
	if err != nil {
		return nil, err
	}

	res, err := io.ReadAll(out)
	if err != nil {
		return nil, err
	}

	return res, cmd.Wait()
}

type interpreter struct {
	executable string
	args       []string
}

func findInterpreter() (*interpreter, error) {
	// Lookup for bash executable first
	out, err := exec.LookPath("bash")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, err
	}

	if out != "" {
		return &interpreter{
			executable: out,
			args:       []string{"-c"},
		}, nil
	}

	// Lookup for sh executable
	out, err = exec.LookPath("sh")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, err
	}

	if out != "" {
		return &interpreter{
			executable: out,
			args:       []string{"-c"},
		}, nil
	}

	// Lookup for PowerShell executable
	out, err = exec.LookPath("powershell")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, err
	}

	if out != "" {
		return &interpreter{
			executable: out,
			args:       []string{"-Command"},
		}, nil
	}

	// Lookup for CMD executable
	out, err = exec.LookPath("cmd")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, err
	}

	if out != "" {
		return &interpreter{
			executable: out,
			args:       []string{"/C"},
		}, nil
	}

	return nil, errors.New("no interpreter found")
}
