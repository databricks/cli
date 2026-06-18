package aircmd

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStubCommandsReturnNotImplemented asserts each unimplemented subcommand
// fails with a "not implemented" error. Drop a command here once it lands.
func TestStubCommandsReturnNotImplemented(t *testing.T) {
	stubs := map[string]*cobra.Command{
		"run":            newRunCommand(),
		"list":           newListCommand(),
		"logs":           newLogsCommand(),
		"cancel":         newCancelCommand(),
		"register-image": newRegisterImageCommand(),
	}

	for name, cmd := range stubs {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, cmd.RunE, "command should define RunE")
			err := cmd.RunE(cmd, nil)
			assert.EqualError(t, err, fmt.Sprintf("`air %s` is not implemented yet", name))
		})
	}
}
