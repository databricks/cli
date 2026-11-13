package root

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// protocolTestCommand is a command annotated as writing a protocol to stdout. It records whether it ran.
func protocolTestCommand(ran *bool) *cobra.Command {
	return &cobra.Command{
		Use:         "serve",
		Annotations: map[string]string{StdoutCarriesProtocol: "true"},
		RunE: func(*cobra.Command, []string) error {
			*ran = true
			return nil
		},
	}
}

// keepDefaultLogger restores the slog default that Execute replaces when it opens a log file.
func keepDefaultLogger(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
}

// useSeparateStderr points os.Stderr at a file of its own, as an agent gives the server separate stdout and stderr.
// Under `go test -json` os.Stderr is os.Stdout, which would make the stdout cases below look like stderr.
func useSeparateStderr(t *testing.T) {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "stderr.txt"))
	require.NoError(t, err)
	prev := os.Stderr
	os.Stderr = f
	t.Cleanup(func() {
		os.Stderr = prev
		f.Close()
	})
}

func TestExecuteRefusesStdoutLogForProtocolCommand(t *testing.T) {
	keepDefaultLogger(t)
	useSeparateStderr(t)
	for _, logFile := range []string{"stdout", "STDOUT", "/dev/stdout"} {
		t.Run(logFile, func(t *testing.T) {
			if logFile == "/dev/stdout" {
				if _, err := os.Stat(logFile); err != nil {
					t.Skip("this OS has no /dev/stdout")
				}
			}
			ran := false
			cmd := New(t.Context())
			cmd.AddCommand(protocolTestCommand(&ran))
			cmd.SetArgs([]string{"serve", "--log-file", logFile, "--log-level", "debug"})
			cmd.SetErr(io.Discard)

			err := Execute(t.Context(), cmd)
			require.ErrorContains(t, err, "writes protocol messages to stdout")
			assert.False(t, ran)
		})
	}
}

func TestExecuteAllowsFileLogForProtocolCommand(t *testing.T) {
	keepDefaultLogger(t)
	useSeparateStderr(t)
	logFile := filepath.Join(t.TempDir(), "serve.log")
	ran := false
	cmd := New(t.Context())
	cmd.AddCommand(protocolTestCommand(&ran))
	cmd.SetArgs([]string{"serve", "--log-file", logFile, "--log-level", "debug"})
	cmd.SetErr(io.Discard)

	require.NoError(t, Execute(t.Context(), cmd))
	assert.True(t, ran)
}

// writesToStdout compares the log file with the stdout and stderr it is given, so a path that is stdout under
// another name is refused. stderr is never refused. The streams are pipes of the test's own, not the process's.
func TestWritesToStdout(t *testing.T) {
	stdout := newPipe(t)
	stderr := newPipe(t)
	other := newPipe(t)
	file, err := os.Create(filepath.Join(t.TempDir(), "run.log"))
	require.NoError(t, err)
	t.Cleanup(func() { file.Close() })
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { null.Close() })

	tests := []struct {
		name string
		w    io.Writer
		want bool
	}{
		{name: "stdout", w: stdout, want: true},
		{name: "stderr", w: stderr, want: false},
		{name: "another pipe", w: other, want: false},
		{name: "regular file", w: file, want: false},
		{name: "null device", w: null, want: false},
		{name: "not a file", w: io.Discard, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := writesToStdout(tc.w, stdout, stderr)
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}
}

// When stdout and stderr are one file, as on a terminal, a log file that is that file is not refused: the protocol
// already shares it with the diagnostics. A second handle to it is that file, so it is not refused either.
func TestWritesToStdoutAllowsStderrFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "terminal.txt")
	shared, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { shared.Close() })
	second, err := os.OpenFile(path, os.O_WRONLY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { second.Close() })

	ok, err := writesToStdout(second, shared, shared)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = writesToStdout(shared, shared, shared)
	require.NoError(t, err)
	assert.False(t, ok)
}

// newPipe returns the write end of a pipe that the test closes.
func newPipe(t *testing.T) *os.File {
	t.Helper()
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		pr.Close()
		pw.Close()
	})
	return pw
}

func TestCarriesProtocol(t *testing.T) {
	assert.True(t, carriesProtocol(protocolTestCommand(new(bool))))
	assert.False(t, carriesProtocol(&cobra.Command{Use: "serve"}))
}
