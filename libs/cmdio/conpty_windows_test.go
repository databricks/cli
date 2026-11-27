package cmdio

import (
	"fmt"
	"os"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/env"
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

func TestNewCapabilitiesMSYSPipeStdin(t *testing.T) {
	// End to end through newCapabilities: an MSYS2 pty pipe on stdin, with a
	// terminal on stderr, allows interactive output but not prompts.
	id := fmt.Sprintf("%016x", os.Getpid()+2)
	in := openNamedPipe(t, "msys-"+id+"-pty0-from-master")
	errw := openNamedPipe(t, "msys-"+id+"-pty0-to-master")
	// Pin the environment so a developer's NO_COLOR or TERM=dumb can't leak in.
	ctx := env.Set(env.Set(t.Context(), "NO_COLOR", ""), "TERM", "")
	c := newCapabilities(ctx, in, errw, errw)
	assert.True(t, c.stdinIsMSYSPipe)
	assert.True(t, c.SupportsInteractive())
	assert.False(t, c.SupportsPrompt())

	// The same pipe on stdout or stderr doesn't affect prompt support.
	c = newCapabilities(ctx, FakeTTYReader(nil), errw, errw)
	assert.False(t, c.stdinIsMSYSPipe)
	assert.True(t, c.SupportsPrompt())
}

func TestNewRendererRaisesMSYSPipeToANSI(t *testing.T) {
	// termenv reads NO_COLOR from the process environment.
	t.Setenv("NO_COLOR", "")
	// termenv can't recognize an MSYS2 pty pipe as a terminal, but cmdio
	// does, so the renderer must still emit color there.
	f := openNamedPipe(t, fmt.Sprintf("msys-%016x-pty0-to-master", os.Getpid()+1))
	ctx := InContext(t.Context(), &cmdIO{capabilities: Capabilities{stderrIsTTY: true}})
	r, color := NewRenderer(ctx, f)
	assert.True(t, color)
	assert.Equal(t, colorprofile.ANSI, r.Profile())
	assert.NotEqual(t, "hi", r.NewStyle().Foreground(lipgloss.Color("10")).Render("hi"))
}

func TestNewRendererMSYSPipeHonorsCLICOLOR(t *testing.T) {
	// termenv reads NO_COLOR from the process environment.
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "0")
	f := openNamedPipe(t, fmt.Sprintf("msys-%016x-pty1-to-master", os.Getpid()+3))
	ctx := InContext(t.Context(), &cmdIO{capabilities: Capabilities{stderrIsTTY: true}})
	r, _ := NewRenderer(ctx, f)
	assert.Equal(t, colorprofile.Ascii, r.Profile())
}
