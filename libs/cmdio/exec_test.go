package cmdio

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutorWithSimpleInput(t *testing.T) {
	executor := NewCommandExecutor(".")
	out, err := executor.Exec(context.Background(), "echo 'Hello'")
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Hello\n", string(out))
}

func TestExecutorWithComplexInput(t *testing.T) {
	executor := NewCommandExecutor(".")
	out, err := executor.Exec(context.Background(), "echo 'Hello' && echo 'World'")
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Hello\nWorld\n", string(out))
}

func TestExecutorWithInvalidCommand(t *testing.T) {
	executor := NewCommandExecutor(".")
	out, err := executor.Exec(context.Background(), "invalid-command")
	assert.Error(t, err)

	if runtime.GOOS == "windows" {
		assert.Contains(t, string(out), "'invalid-command' is not recognized")
	} else {
		assert.Contains(t, string(out), "invalid-command: command not found")
	}
}

func TestExecutorWithInvalidCommandWithWindowsLikePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	executor := NewCommandExecutor(".")
	out, err := executor.Exec(context.Background(), `"C:\Program Files\invalid-command.exe"`)
	assert.Error(t, err)
	assert.Contains(t, string(out), "'C:\\Program Files\\invalid-command.exe' is not recognized")
}
