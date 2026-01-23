package tui

import (
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newSpinner() *cobra.Command {
	return &cobra.Command{
		Use:   "spinner",
		Short: "Test the cmdio spinner component",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			sp := cmdio.NewSpinner(ctx)

			// Test various status messages
			messages := []struct {
				text     string
				duration time.Duration
			}{
				{"Initializing...", time.Second},
				{"Loading configuration", time.Second},
				{"Connecting to workspace", time.Second},
				{"Processing files", time.Second},
				{"Finalizing", time.Second},
			}

			for _, msg := range messages {
				sp.Update(msg.text)
				time.Sleep(msg.duration)
			}

			sp.Close()

			cmdio.LogString(ctx, "✓ Spinner test complete")
		},
	}
}
