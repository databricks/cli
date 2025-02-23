package python

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// GetExecutable gets appropriate python binary name for the platform
func GetExecutable() string {
	// On Windows when virtualenv is created, the <env>/Scripts directory
	// contains python.exe but no python3.exe.
	// Most installers (e.g. the ones from python.org) only install python.exe and not python3.exe

	if runtime.GOOS == "windows" {
		return "python"
	} else {
		return "python3"
	}
}

// DetectExecutable looks up the path to the python3 executable from the PATH
// environment variable.
//
// If virtualenv is activated, executable from the virtualenv is returned,
// because activating virtualenv adds python3 executable on a PATH.
//
// If python3 executable is not found on the PATH, the interpreter with the
// least version that satisfies minimal 3.8 version is returned, e.g.
// python3.10.
func DetectExecutable(ctx context.Context) (string, error) {
	// TODO: add a shortcut if .python-version file is detected somewhere in
	// the parent directory tree.
	//
	// See https://github.com/pyenv/pyenv#understanding-python-version-selection

	return exec.LookPath(GetExecutable())
}

// DetectVEnvExecutable returns the path to the python3 executable inside venvPath,
// that is not necessarily activated.
//
// If virtualenv is not created, or executable doesn't exist, the error is returned.
func DetectVEnvExecutable(venvPath string) (string, error) {
	interpreterPath := filepath.Join(venvPath, "bin", "python3")
	if runtime.GOOS == "windows" {
		interpreterPath = filepath.Join(venvPath, "Scripts", "python.exe")
	}

	if _, err := os.Stat(interpreterPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("can't find %q, check if virtualenv is created", interpreterPath)
		} else {
			return "", fmt.Errorf("can't find %q: %w", interpreterPath, err)
		}
	}

	// pyvenv.cfg must be always present in correctly configured virtualenv,
	// read more in https://snarky.ca/how-virtual-environments-work/
	pyvenvPath := filepath.Join(venvPath, "pyvenv.cfg")
	if _, err := os.Stat(pyvenvPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("expected %q to be virtualenv, but pyvenv.cfg is missing", venvPath)
		} else {
			return "", fmt.Errorf("can't find %q: %w", pyvenvPath, err)
		}
	}

	return interpreterPath, nil
}
