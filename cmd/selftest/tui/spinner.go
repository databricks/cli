package tui

import (
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newSpinnerCmd() *cobra.Command {
	var elapsed bool
	cmd := &cobra.Command{
		Use:   "spinner",
		Short: "cmdio.NewSpinner (progress indicator)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var opts []cmdio.SpinnerOption
			if elapsed {
				opts = append(opts, cmdio.WithElapsedTime())
			}
			sp := cmdio.NewSpinner(ctx, opts...)

			for _, msg := range spinnerMessages {
				sp.Update(msg.text)
				time.Sleep(msg.duration)
			}

			sp.Close()

			cmdio.LogString(ctx, "Spinner test complete")
			return nil
		},
	}
	cmd.Flags().BoolVar(&elapsed, "elapsed", false, "show an MM:SS elapsed-time prefix on the spinner")
	return cmd
}
