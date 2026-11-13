package root

import (
	"io"
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

func TestExecuteRefusesStdoutLogForProtocolCommand(t *testing.T) {
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
	logFile := filepath.Join(t.TempDir(), "serve.log")
	ran := false
	cmd := New(t.Context())
	cmd.AddCommand(protocolTestCommand(&ran))
	cmd.SetArgs([]string{"serve", "--log-file", logFile, "--log-level", "debug"})
	cmd.SetErr(io.Discard)

	require.NoError(t, Execute(t.Context(), cmd))
	assert.True(t, ran)
}

func TestCarriesProtocol(t *testing.T) {
	assert.True(t, carriesProtocol(protocolTestCommand(new(bool))))
	assert.False(t, carriesProtocol(&cobra.Command{Use: "serve"}))
}
