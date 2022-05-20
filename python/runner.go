package python

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func PyInline(ctx context.Context, inlinePy string) (string, error) {
	return Py(ctx, "-c", TrimLeadingWhitespace(inlinePy))
}

func Py(ctx context.Context, script string, args ...string) (string, error) {
	py, err := detectExecutable(ctx)
	if err != nil {
		return "", err
	}
	out, err := execAndPassErr(ctx, py, append([]string{script}, args...)...)
	if err != nil {
		// current error message chain is longer:
		// failed to call {pyExec} __non_existing__.py: {pyExec}: can't open
		// ... file '{pwd}/__non_existing__.py': [Errno 2] No such file or directory"
		// probably we'll need to make it shorter:
		// can't open file '$PWD/__non_existing__.py': [Errno 2] No such file or directory
		return "", err
	}
	return trimmedS(out), nil
}

func createVirtualEnv(ctx context.Context) error {
	_, err := Py(context.Background(), "-m", "venv", ".venv")
	return err
}

// python3 -m build -w
// https://packaging.python.org/en/latest/tutorials/packaging-projects/
func detectVirtualEnv() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
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
	detector := "which"
	if runtime.GOOS == "windows" {
		detector = "where.exe"
	}
	out, err := execAndPassErr(ctx, detector, "python3")
	if err != nil {
		return "", err
	}
	pyExec = trimmedS(out)
	return pyExec, nil
}

func execAndPassErr(ctx context.Context, name string, args ...string) ([]byte, error) {
	// TODO: move out to a separate package, once we have Maven integration
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return out, nicerErr(err)
}

func nicerErr(err error) error {
	if err == nil {
		return nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		errMsg := trimmedS(ee.Stderr)
		if errMsg == "" {
			errMsg = err.Error()
		}
		return errors.New(errMsg)
	}
	return err
}

func trimmedS(bytes []byte) string {
	return strings.Trim(string(bytes), "\n\r")
}

// TrimLeadingWhitespace removes leading whitespace
// function copied from Databricks Terraform provider
func TrimLeadingWhitespace(commandStr string) (newCommand string) {
	lines := strings.Split(strings.ReplaceAll(commandStr, "\t", "    "), "\n")
	leadingWhitespace := 1<<31 - 1
	for _, line := range lines {
		for pos, char := range line {
			if char == ' ' || char == '\t' {
				continue
			}
			// first non-whitespace character
			if pos < leadingWhitespace {
				leadingWhitespace = pos
			}
			// is not needed further
			break
		}
	}
	for i := 0; i < len(lines); i++ {
		if lines[i] == "" || strings.Trim(lines[i], " \t") == "" {
			continue
		}
		if len(lines[i]) < leadingWhitespace {
			newCommand += lines[i] + "\n" // or not..
		} else {
			newCommand += lines[i][leadingWhitespace:] + "\n"
		}
	}
	return
}
