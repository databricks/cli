package python

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/bricks/lib/spawn"
	"github.com/databricks/databricks-sdk-go/service/commands"
)

func PyInline(ctx context.Context, inlinePy string) (string, error) {
	return Py(ctx, "-c", commands.TrimLeadingWhitespace(inlinePy))
}

// Py calls system-detected Python3 executable
func Py(ctx context.Context, args ...string) (string, error) {
	py, err := detectExecutable(ctx)
	if err != nil {
		return "", err
	}
	out, err := spawn.ExecAndPassErr(ctx, py, args...)
	if err != nil {
		// current error message chain is longer:
		// failed to call {pyExec} __non_existing__.py: {pyExec}: can't open
		// ... file '{pwd}/__non_existing__.py': [Errno 2] No such file or directory"
		// probably we'll need to make it shorter:
		// can't open file '$PWD/__non_existing__.py': [Errno 2] No such file or directory
		return "", err
	}
	return strings.Trim(string(out), "\n\r"), nil
}

func createVirtualEnv(ctx context.Context) error {
	_, err := Py(context.Background(), "-m", "venv", ".venv")
	return err
}

// python3 -m build -w
// https://packaging.python.org/en/latest/tutorials/packaging-projects/
func detectVirtualEnv(wd string) (string, error) {
	wdf, err := os.Open(wd)
	if err != nil {
		return "", err
	}
	files, err := wdf.ReadDir(0)
	if err != nil {
		return "", err
	}
	for _, v := range files {
		if !v.IsDir() {
			continue
		}
		candidate := fmt.Sprintf("%s/%s", wd, v.Name())
		_, err = os.Stat(fmt.Sprintf("%s/pyvenv.cfg", candidate))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return candidate, nil
	}
	return "", nil
}

var pyExec string

func detectExecutable(ctx context.Context) (string, error) {
	if pyExec != "" {
		return pyExec, nil
	}
	detected, err := spawn.DetectExecutable(ctx, "python3")
	if err != nil {
		return "", err
	}
	pyExec = detected
	return pyExec, nil
}
