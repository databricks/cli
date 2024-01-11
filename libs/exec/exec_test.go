package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
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

func testExecutorWithShell(t *testing.T, shell string) {
	p, err := osexec.LookPath(shell)
	if err != nil {
		if errors.Is(err, osexec.ErrNotFound) {
			switch runtime.GOOS {
			case "windows":
				if shell == "cmd" {
					// We must find `cmd.exe` on Windows.
					t.Fatal("cmd.exe not found")
				}
			default:
				if shell == "bash" || shell == "sh" {
					// We must find `bash` or `sh` on other operating systems.
					t.Fatal("bash or sh not found")
				}
			}
			t.Skipf("shell %s not found", shell)
		}
		t.Fatal(err)
	}

	// Create temporary directory with only the shell executable in the PATH.
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	if runtime.GOOS == "windows" {
		os.Symlink(p, fmt.Sprintf("%s/%s.exe", tmpDir, shell))
	} else {
		os.Symlink(p, fmt.Sprintf("%s/%s", tmpDir, shell))
	}

	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)
	out, err := executor.Exec(context.Background(), "echo 'Hello'")
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Hello\n", string(out))
}

func TestExecutorWithDifferentShells(t *testing.T) {
	for _, shell := range []string{"bash", "sh", "cmd"} {
		t.Run(shell, func(t *testing.T) {
			testExecutorWithShell(t, shell)
		})
	}
}

func TestExecutorNoShellFound(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := NewCommandExecutor(".")
	assert.ErrorContains(t, err, "no shell found")
}

func TestExecutorCleanupsTempFiles(t *testing.T) {
	executor, err := NewCommandExecutor(".")
	assert.NoError(t, err)

	ec, err := executor.shell.prepare("echo 'Hello'")
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
		go func(cmd Command, i int) {
			defer wg.Done()

			stdout := cmd.Stdout()
			out, err := io.ReadAll(stdout)
			assert.NoError(t, err)

			err = cmd.Wait()
			assert.NoError(t, err)

			assert.Equal(t, fmt.Sprintf("Hello %d\n", i), string(out))
		}(cmd, i)
		assert.NoError(t, err)
	}

	wg.Wait()
}
