package tui

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	var defaultVal string
	cmd := &cobra.Command{
		Use:   "ask",
		Short: "cmdio.Ask (single-line text input with optional default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ans, err := cmdio.Ask(ctx, "Enter a value", defaultVal)
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, "Entered: "+ans)
			return nil
		},
	}
	cmd.Flags().StringVar(&defaultVal, "default", "", "default returned if the user just presses Enter")
	return cmd
}

func newAskYesOrNoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask-yes-no",
		Short: "cmdio.AskYesOrNo (yes/no question)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ans, err := cmdio.AskYesOrNo(ctx, "Continue")
			if err != nil {
				return err
			}
			if ans {
				cmdio.LogString(ctx, "Answer: yes")
			} else {
				cmdio.LogString(ctx, "Answer: no")
			}
			return nil
		},
	}
}
