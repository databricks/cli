package onechat

import "github.com/spf13/cobra"

// NewOneChatCmd creates the parent "onechat" command group.
func NewOneChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "onechat",
		Short:  "Ask data questions via Databricks One Chat",
		Hidden: true,
	}

	cmd.AddCommand(newAskCmd())

	return cmd
}
