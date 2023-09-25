package python

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecAndPassError(t *testing.T) {

	// remove this once equivalent tests for windows have been set up
	// or this test has been fixed for windows
	// date: 28 Nov 2022
	if runtime.GOOS == "windows" {
		t.Skip("skipping temperorilty to make windows unit tests green")
	}

	_, err := execAndPassErr(context.Background(), "which", "__non_existing__")
	assert.EqualError(t, err, "which __non_existing__:  exit status 1")
}

func TestDetectPython(t *testing.T) {
	pyExec = ""
	py, err := DetectExecutable(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, py, "python3")
}

func TestDetectPythonCache(t *testing.T) {
	pyExec = "abc"
	py, err := DetectExecutable(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "abc", py)
	pyExec = ""
}

func TestDetectVirtualEnvFalse(t *testing.T) {
	venvDir, err := detectVirtualEnv()
	assert.NoError(t, err)
	assert.Equal(t, "", venvDir)
}

func TestGetFirstMatch(t *testing.T) {
	matches := "C:\\hostedtoolcache\\windows\\Python\\3.9.13\\x64\\python3.exe\r\nC:\\ProgramData\\Chocolatey\\bin\\python3.exe"
	assert.Equal(t, getFirstMatch(matches), "C:\\hostedtoolcache\\windows\\Python\\3.9.13\\x64\\python3.exe")
}

func TestMakeDetectableVenv(t *testing.T) {
	var temp string
	defer testTempdir(t, &temp)()

	// TODO: rewrite with t.TempDir() and arguments
	err := createVirtualEnv(context.Background())
	assert.NoError(t, err)

	venv, err := detectVirtualEnv()
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s/.venv", temp), venv)
}

func testTempdir(t *testing.T, dir *string) func() {
	wd, _ := os.Getwd()
	temp, err := os.MkdirTemp(os.TempDir(), "brickstest")
	assert.NoError(t, err)
	os.Chdir(temp)
	wd2, _ := os.Getwd()
	*dir = wd2
	return func() {
		os.Chdir(wd)
		os.RemoveAll(temp)
	}
}

func TestPyError(t *testing.T) {
	_, err := Py(context.Background(), "__non_existing__.py")
	assert.Contains(t, err.Error(), "can't open file")
}

func TestPyInline(t *testing.T) {
	hello, err := PyInline(context.Background(), "print('Hello, world!')")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", hello)
}

func TestPyInlineStderr(t *testing.T) {
	DetectExecutable(context.Background())
	inline := "import sys; sys.stderr.write('___msg___'); sys.exit(1)"
	_, err := PyInline(context.Background(), inline)
	assert.ErrorContains(t, err, "___msg___")
}
