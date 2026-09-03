package aircmd

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestNewRegistersAllSubcommands asserts the `air` command wires up every
// expected subcommand, so none is accidentally dropped from New.
func TestNewRegistersAllSubcommands(t *testing.T) {
	registered := make(map[string]bool)
	for _, c := range New().Commands() {
		registered[c.Name()] = true
	}

	want := []string{"run", "get", "list", "logs", "cancel", "register-image", "convert-to-dabs"}
	for _, name := range want {
		assert.True(t, registered[name], "subcommand %q is not registered", name)
	}
	assert.Len(t, registered, len(want), "unexpected number of subcommands")
}

func TestRunErrorIncludesDebugTip(t *testing.T) {
	originalErr := errors.New("failed")
	runCommand := &cobra.Command{
		Use: "run [arg]",
		RunE: func(cmd *cobra.Command, args []string) error {
			return originalErr
		},
	}
	airCommand := &cobra.Command{Use: "air"}
	airCommand.AddCommand(runCommand)
	wrapRunErrorsWithDebugTip(airCommand)
	experimentalCommand := &cobra.Command{Use: "experimental"}
	experimentalCommand.AddCommand(airCommand)
	rootCommand := &cobra.Command{Use: "databricks"}
	rootCommand.PersistentFlags().Bool("debug", false, "")
	rootCommand.AddCommand(experimentalCommand)

	err := runCommand.RunE(runCommand, []string{"secret-value"})

	assert.ErrorIs(t, err, originalErr)
	assert.Contains(t, err.Error(), "use the --debug flag")
	assert.Contains(t, err.Error(), "databricks --debug experimental air run …")
	assert.NotContains(t, err.Error(), "secret-value")
}

func TestRunErrorOmitsDebugTipWhenDebugEnabled(t *testing.T) {
	originalErr := errors.New("failed")
	runCommand := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return originalErr
		},
	}
	airCommand := &cobra.Command{Use: "air"}
	airCommand.AddCommand(runCommand)
	wrapRunErrorsWithDebugTip(airCommand)
	rootCommand := &cobra.Command{Use: "databricks"}
	rootCommand.PersistentFlags().Bool("debug", true, "")
	rootCommand.AddCommand(airCommand)

	err := runCommand.RunE(runCommand, nil)

	assert.Same(t, originalErr, err)
	assert.NotContains(t, err.Error(), "use the --debug flag")
}
