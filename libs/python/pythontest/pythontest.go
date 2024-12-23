package pythontest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

type VenvOpts struct {
	// input
	PythonVersion    string
	skipVersionCheck bool

	// input/output
	Dir  string
	Name string

	// output:
	// Absolute path to venv
	EnvPath string

	// Absolute path to venv/bin or venv/Scripts, depending on OS
	BinPath string

	// Absolute path to python binary
	PythonExe string
}

func CreatePythonEnv(opts *VenvOpts) error {
	if opts == nil || opts.PythonVersion == "" {
		return errors.New("PythonVersion must be provided")
	}
	if opts.Name == "" {
		opts.Name = testutil.RandomName("test-venv-")
	}

	cmd := exec.Command("uv", "venv", opts.Name, "--python", opts.PythonVersion, "--seed", "-q")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = opts.Dir
	err := cmd.Run()
	if err != nil {
		return err
	}

	opts.EnvPath, err = filepath.Abs(filepath.Join(opts.Dir, opts.Name))
	if err != nil {
		return err
	}

	_, err = os.Stat(opts.EnvPath)
	if err != nil {
		return fmt.Errorf("cannot stat EnvPath %s: %s", opts.EnvPath, err)
	}

	if runtime.GOOS == "windows" {
		// https://github.com/pypa/virtualenv/commit/993ba1316a83b760370f5a3872b3f5ef4dd904c1
		opts.BinPath = filepath.Join(opts.EnvPath, "Scripts")
		opts.PythonExe = filepath.Join(opts.BinPath, "python.exe")
	} else {
		opts.BinPath = filepath.Join(opts.EnvPath, "bin")
		opts.PythonExe = filepath.Join(opts.BinPath, "python3")
	}

	_, err = os.Stat(opts.BinPath)
	if err != nil {
		return fmt.Errorf("cannot stat BinPath %s: %s", opts.BinPath, err)
	}

	_, err = os.Stat(opts.PythonExe)
	if err != nil {
		return fmt.Errorf("cannot stat PythonExe %s: %s", opts.PythonExe, err)
	}

	if !opts.skipVersionCheck {
		cmd := exec.Command(opts.PythonExe, "--version")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to run %s --version: %s", opts.PythonExe, err)
		}
		outString := string(out)
		expectVersion := "Python " + opts.PythonVersion
		if !strings.HasPrefix(outString, expectVersion) {
			return fmt.Errorf("Unexpected output from %s --version: %v (expected %v)", opts.PythonExe, outString, expectVersion)
		}
	}

	return nil
}

func RequireActivatedPythonEnv(t *testing.T, ctx context.Context, opts *VenvOpts) {
	err := CreatePythonEnv(opts)
	require.NoError(t, err)
	require.DirExists(t, opts.BinPath)

	newPath := fmt.Sprintf("%s%c%s", opts.BinPath, os.PathListSeparator, os.Getenv("PATH"))
	t.Setenv("PATH", newPath)
}
