package cmdio

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

// openNamedPipe creates a named pipe server and returns a client handle to
// it, mimicking the handles an MSYS2 runtime without ConPTY passes to native
// programs.
func openNamedPipe(t *testing.T, name string) *os.File {
	t.Helper()
	path, err := windows.UTF16PtrFromString(`\\.\pipe\` + name)
	require.NoError(t, err)

	server, err := windows.CreateNamedPipe(path,
		windows.PIPE_ACCESS_DUPLEX, windows.PIPE_TYPE_BYTE|windows.PIPE_WAIT,
		1, 4096, 4096, 0, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = windows.CloseHandle(server) })

	client, err := windows.CreateFile(path,
		windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
	require.NoError(t, err)
	f := os.NewFile(uintptr(client), name)
	t.Cleanup(func() { f.Close() })
	return f
}

func TestIsMSYSPipeWindows(t *testing.T) {
	id := fmt.Sprintf("%016x", os.Getpid())

	for _, name := range []string{
		"msys-" + id + "-pty0-from-master",
		"msys-" + id + "-pty0-to-master",
		"cygwin-" + id + "-pty1-from-master",
	} {
		t.Run(name, func(t *testing.T) {
			f := openNamedPipe(t, name)
			assert.True(t, isMSYSPipe(f))
			// It still counts as a TTY for output and line input.
			assert.True(t, isTTY(f))
		})
	}

	t.Run("other named pipe", func(t *testing.T) {
		f := openNamedPipe(t, "databricks-cli-test-"+id)
		assert.False(t, isMSYSPipe(f))
		assert.False(t, isTTY(f))
	})
}
