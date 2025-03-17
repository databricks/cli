package bundle

import (
	"errors"

	"github.com/spf13/cobra"
)

func newShowCommand(hidden bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a preview for a table",
		Long:  `Show a preview for a table.`,

		// We're not ready to expose this command until we specify its semantics.
		Hidden: hidden,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.New("TODO")
	}

	return cmd
}
