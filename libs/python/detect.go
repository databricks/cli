package python

import (
	"context"
	"errors"
	"os/exec"
)

func DetectExecutable(ctx context.Context) (string, error) {
	// TODO: add a shortcut if .python-version file is detected somewhere in
	// the parent directory tree.
	//
	// See https://github.com/pyenv/pyenv#understanding-python-version-selection
	out, err := exec.LookPath("python3")
	// most of the OS'es have python3 in $PATH, but for those which don't,
	// we perform the latest version lookup
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return "", err
	}
	if out != "" {
		return out, nil
	}
	// otherwise, detect all interpreters and pick the least that satisfies
	// minimal version requirements
	all, err := DetectInterpreters(ctx)
	if err != nil {
		return "", err
	}
	interpreter, err := all.AtLeast("3.8")
	if err != nil {
		return "", err
	}
	return interpreter.Path, nil
}
