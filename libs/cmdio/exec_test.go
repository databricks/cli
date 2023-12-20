package cmdio

import (
	"context"
	"io"
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
	assert.Contains(t, string(out), "invalid-command: command not found")
}

func TestExecutorWithInvalidCommandWithWindowsLikePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	executor := NewCommandExecutor(".")
	out, err := executor.Exec(context.Background(), `"C:\Program Files\invalid-command.exe"`)
	assert.Error(t, err)
	assert.Contains(t, string(out), "C:\\Program Files\\invalid-command.exe: No such file or directory")
}

func TestFindBashExecutableNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	executable, err := findBashExecutable(`echo "Hello from bash"`)
	assert.NoError(t, err)
	assert.NotEmpty(t, executable)

	e := NewCommandExecutor(".")
	cmd, reader, err := e.start(context.Background(), executable)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.NotNil(t, reader)

	out, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, "Hello from bash\n", string(out))
	cmd.Wait()
}

func TestFindCmdExecutable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	executable, err := findCmdExecutable(`echo "Hello from cmd"`)
	assert.NoError(t, err)
	assert.NotEmpty(t, executable)

	e := NewCommandExecutor(".")
	cmd, reader, err := e.start(context.Background(), executable)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.NotNil(t, reader)

	out, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Contains(t, string(out), "Hello from cmd")
	cmd.Wait()
}
