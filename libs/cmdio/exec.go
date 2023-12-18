package cmdio

import (
	"context"
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

	// TODO: switch to process.Background(...)
	cmd := exec.CommandContext(ctx, interpreter, "-c", command)
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

func findInterpreter() (string, error) {
	// At the moment we just return 'sh' on all platforms and use it to execute scripts
	return "sh", nil
}
