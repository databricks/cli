package python

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/databricks/bricks/lib/spawn"
	"github.com/stretchr/testify/assert"
)

func TestDetectPython(t *testing.T) {
	pyExec = ""
	py, err := detectExecutable(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, py, "python3")
}

func TestDetectPythonCache(t *testing.T) {
	pyExec = "abc"
	py, err := detectExecutable(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "abc", py)
	pyExec = ""
}

func TestDetectVirtualEnvFalse(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	venvDir, err := detectVirtualEnv(wd)
	assert.NoError(t, err)
	assert.Equal(t, "", venvDir)
}

func TestMakeDetectableVenv(t *testing.T) {
	temp := t.TempDir()
	ctx := spawn.WithRoot(context.Background(), temp)

	err := createVirtualEnv(ctx)
	assert.NoError(t, err)

	venv, err := detectVirtualEnv(temp)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s/.venv", temp), venv)
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
	detectExecutable(context.Background())
	inline := "import sys; sys.stderr.write('___msg___'); sys.exit(1)"
	_, err := PyInline(context.Background(), inline)
	assert.EqualError(t, err, "___msg___")
}
