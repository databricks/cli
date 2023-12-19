package cmdio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Executor struct {
	dir         string
	scriptFiles []string
}

func NewCommandExecutor(dir string) *Executor {
	return &Executor{
		dir:         dir,
		scriptFiles: nil,
	}
}

func (e *Executor) StartCommand(ctx context.Context, command string) (*exec.Cmd, io.Reader, error) {
	interpreter, err := wrapInShellCall(command)

	if err != nil {
		return nil, nil, err
	}

	e.scriptFiles = append(e.scriptFiles, interpreter.scriptFile)
	cmd := exec.CommandContext(ctx, interpreter.executable, interpreter.args...)
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

	defer e.Cleanup()
	return res, cmd.Wait()
}

func (e *Executor) Cleanup() {
	if e.scriptFiles != nil {
		for _, file := range e.scriptFiles {
			os.Remove(file)
		}
	}
	e.scriptFiles = nil
}

type interpreter struct {
	executable string
	args       []string
	scriptFile string
}

func wrapInShellCall(command string) (*interpreter, error) {
	// Lookup for bash executable first (Linux, MacOS, maybe Windows)
	out, err := exec.LookPath("bash")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, err
	}

	if out != "" {
		filename, err := createTempScript(command, ".sh")
		if err != nil {
			return nil, err
		}
		return &interpreter{
			executable: out,
			args:       []string{"-e", filename},
			scriptFile: filename,
		}, nil
	}

	// Lookup for CMD executable (Windows)
	out, err = exec.LookPath("cmd")
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, err
	}

	if out != "" {
		filename, err := createTempScript(command, ".cmd")
		if err != nil {
			return nil, err
		}
		return &interpreter{
			executable: out,
			args:       []string{"/D", "/E:ON", "/V:OFF", "/S", "/C", fmt.Sprintf(`CALL %s`, filename)},
			scriptFile: filename,
		}, nil
	}

	return nil, errors.New("no interpreter found")
}

func createTempScript(command string, extension string) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "cli-exec*"+extension)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = io.WriteString(file, command)
	file.Close()
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}
