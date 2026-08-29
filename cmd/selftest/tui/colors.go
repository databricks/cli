package tui

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newColorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "colors",
		Short: "Print colored text to verify color support",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			swatch := "the quick brown fox jumps over the lazy dog"
			cmdio.LogString(ctx, "red:     "+cmdio.Red(ctx, swatch))
			cmdio.LogString(ctx, "green:   "+cmdio.Green(ctx, swatch))
			cmdio.LogString(ctx, "yellow:  "+cmdio.Yellow(ctx, swatch))
			cmdio.LogString(ctx, "blue:    "+cmdio.Blue(ctx, swatch))
			cmdio.LogString(ctx, "cyan:    "+cmdio.Cyan(ctx, swatch))
			cmdio.LogString(ctx, "hiblack: "+cmdio.HiBlack(ctx, swatch))
			cmdio.LogString(ctx, "hiblue:  "+cmdio.HiBlue(ctx, swatch))
			return nil
		},
	}
}
