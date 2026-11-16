package tui

import (
	"fmt"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newSpinnerCmd() *cobra.Command {
	var (
		elapsed  bool
		duration time.Duration
	)
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
				d := msg.duration
				if duration > 0 {
					d = duration / time.Duration(len(spinnerMessages))
				}
				time.Sleep(d)
			}

			sp.Close()

			cmdio.LogString(ctx, "Spinner test complete")
			return nil
		},
	}
	cmd.Flags().BoolVar(&elapsed, "elapsed", false, "show an MM:SS elapsed-time prefix on the spinner")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if duration < 0 {
			return fmt.Errorf("--duration must not be negative, got %s", duration)
		}
		return nil
	}
	cmd.Flags().DurationVar(&duration, "duration", 0, "total time the spinner runs, split across its messages (0 keeps the built-in per-message timing)")
	return cmd
}
