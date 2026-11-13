package root

import (
	"fmt"
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

// TestExecuteClosesLogFileAndRestoresLogger checks that once Execute returns, the file the command logged to is closed
// and the default logger writes elsewhere. It also checks the panic path, where the command never returns.
func TestExecuteClosesLogFileAndRestoresLogger(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	for _, panics := range []bool{false, true} {
		t.Run(fmt.Sprintf("panics=%v", panics), func(t *testing.T) {
			var slot *logFileSlot
			cmd := New(t.Context())
			cmd.AddCommand(&cobra.Command{
				Use: "serve",
				RunE: func(cmd *cobra.Command, _ []string) error {
					var ok bool
					slot, ok = cmd.Context().Value(logFileKey{}).(*logFileSlot)
					require.True(t, ok)
					if panics {
						panic("boom")
					}
					return nil
				},
			})
			logFile := filepath.Join(t.TempDir(), "run.log")
			cmd.SetArgs([]string{"serve", "--log-file", logFile})
			cmd.SetErr(io.Discard)

			err := Execute(t.Context(), cmd)
			assert.Equal(t, panics, err != nil)
			assert.Equal(t, prev, slog.Default())
			require.NotNil(t, slot)
			assert.ErrorIs(t, slot.file.Close(), os.ErrClosed)
		})
	}
}

func TestExecuteAllowsFileLogForProtocolCommand(t *testing.T) {
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
			ok := writesToStdout(tc.w, stdout, stderr)
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

	ok := writesToStdout(second, shared, shared)
	assert.False(t, ok)

	ok = writesToStdout(shared, shared, shared)
	assert.False(t, ok)
}

// TestWritesToStdoutIgnoresStatFailure checks that a stream that cannot be stat'ed is not stdout: a closed stderr does
// not stop a protocol command, and a closed stdout is not the file a log file names.
func TestWritesToStdoutIgnoresStatFailure(t *testing.T) {
	stdout := newPipe(t)
	closed := newPipe(t)
	require.NoError(t, closed.Close())
	file, err := os.Create(filepath.Join(t.TempDir(), "run.log"))
	require.NoError(t, err)
	t.Cleanup(func() { file.Close() })

	ok := writesToStdout(stdout, stdout, closed)
	assert.True(t, ok, "stdout is still stdout when stderr is closed")

	ok = writesToStdout(file, stdout, closed)
	assert.False(t, ok)

	ok = writesToStdout(file, closed, stdout)
	assert.False(t, ok)
}

// TestExecuteRunsProtocolCommandWithClosedStderr checks that a protocol command without a log file runs when stderr is
// closed, as `2>&-` leaves it.
func TestExecuteRunsProtocolCommandWithClosedStderr(t *testing.T) {
	closed, err := os.Create(filepath.Join(t.TempDir(), "stderr.txt"))
	require.NoError(t, err)
	require.NoError(t, closed.Close())
	prev := os.Stderr
	os.Stderr = closed
	t.Cleanup(func() { os.Stderr = prev })

	ran := false
	cmd := New(t.Context())
	cmd.AddCommand(protocolTestCommand(&ran))
	cmd.SetArgs([]string{"serve"})
	cmd.SetErr(io.Discard)

	require.NoError(t, Execute(t.Context(), cmd))
	assert.True(t, ran)
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
