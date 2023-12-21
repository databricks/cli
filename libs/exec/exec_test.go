package exec

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutorWithSimpleInput(t *testing.T) {
	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)
	out, err := executor.Exec(context.Background(), "echo 'Hello'")
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Hello\n", string(out))
}

func TestExecutorWithComplexInput(t *testing.T) {
	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)
	out, err := executor.Exec(context.Background(), "echo 'Hello' && echo 'World'")
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Hello\nWorld\n", string(out))
}

func TestExecutorWithInvalidCommand(t *testing.T) {
	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)
	out, err := executor.Exec(context.Background(), "invalid-command")
	assert.Error(t, err)
	assert.Contains(t, string(out), "invalid-command: command not found")
}

func TestExecutorWithInvalidCommandWithWindowsLikePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)
	out, err := executor.Exec(context.Background(), `"C:\Program Files\invalid-command.exe"`)
	assert.Error(t, err)
	assert.Contains(t, string(out), "C:\\Program Files\\invalid-command.exe: No such file or directory")
}

func TestFindBashInterpreterNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	interpreter, err := findBashInterpreter()
	assert.NoError(t, err)
	assert.NotEmpty(t, interpreter)

	e, err := NewCommandExecutor(".")
	assert.NoError(t, err)
	e.interpreter = interpreter

	assert.NoError(t, err)
	out, err := e.Exec(context.Background(), `echo "Hello from bash"`)
	assert.NoError(t, err)

	assert.Equal(t, "Hello from bash\n", string(out))
}

func TestFindCmdInterpreter(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	interpreter, err := findCmdInterpreter()
	assert.NoError(t, err)
	assert.NotEmpty(t, interpreter)

	e, err := NewCommandExecutor(".")
	assert.NoError(t, err)
	e.interpreter = interpreter

	assert.NoError(t, err)
	out, err := e.Exec(context.Background(), `echo "Hello from cmd"`)
	assert.NoError(t, err)

	assert.Contains(t, string(out), "Hello from cmd")
}

func TestExecutorCleanupsTempFiles(t *testing.T) {
	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)

	ec, err := executor.interpreter.prepare("echo 'Hello'")
	assert.NoError(t, err)

	cmd, err := executor.start(context.Background(), ec)
	assert.NoError(t, err)

	fileName := ec.args[1]
	assert.FileExists(t, fileName)

	err = cmd.Wait()
	assert.NoError(t, err)
	assert.NoFileExists(t, fileName)
}

func TestMultipleCommandsRunInParrallel(t *testing.T) {
	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)

	const count = 5
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		cmd, err := executor.StartCommand(context.Background(), fmt.Sprintf("echo 'Hello %d'", i))
		// Execute cmd.Wait only when all goroutines are done
		defer cmd.Wait()

		go func(cmd Command, i int) {
			defer wg.Done()

			stdout := cmd.Stdout()
			out, err := io.ReadAll(stdout)
			assert.NoError(t, err)

			assert.Equal(t, fmt.Sprintf("Hello %d\n", i), string(out))
		}(cmd, i)
		assert.NoError(t, err)
	}

	wg.Wait()
}
