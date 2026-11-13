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

func TestExecuteRefusesStdoutLogForProtocolCommand(t *testing.T) {
	keepDefaultLogger(t)
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
	logFile := filepath.Join(t.TempDir(), "serve.log")
	ran := false
	cmd := New(t.Context())
	cmd.AddCommand(protocolTestCommand(&ran))
	cmd.SetArgs([]string{"serve", "--log-file", logFile, "--log-level", "debug"})
	cmd.SetErr(io.Discard)

	require.NoError(t, Execute(t.Context(), cmd))
	assert.True(t, ran)
}

// writesToStdout compares the open log file with the process's stdout, so a path that is stdout under another name is
// refused. stderr is never refused: a user who asks for it gets it, even if it shares a terminal with stdout.
func TestWritesToStdout(t *testing.T) {
	ok, err := writesToStdout(os.Stdout)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = writesToStdout(os.Stderr)
	require.NoError(t, err)
	assert.False(t, ok)

	file, err := os.Create(filepath.Join(t.TempDir(), "run.log"))
	require.NoError(t, err)
	defer file.Close()
	ok, err = writesToStdout(file)
	require.NoError(t, err)
	assert.False(t, ok)

	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	defer pr.Close()
	defer pw.Close()
	ok, err = writesToStdout(pw)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = writesToStdout(io.Discard)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCarriesProtocol(t *testing.T) {
	assert.True(t, carriesProtocol(protocolTestCommand(new(bool))))
	assert.False(t, carriesProtocol(&cobra.Command{Use: "serve"}))
}
