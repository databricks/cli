package tui

import (
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newSecretCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "secret",
		Short: "cmdio.Secret (masked password input)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			value, err := cmdio.Secret(ctx, "Personal access token")
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, fmt.Sprintf("Entered %d characters", len(value)))
			return nil
		},
	}
}
