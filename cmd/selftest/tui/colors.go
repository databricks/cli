package tui

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// swatchCount is the number of blocks per depth row in --lipgloss output.
const swatchCount = 32

func newColorsCmd() *cobra.Command {
	var (
		stream   string
		swatches bool
	)
	cmd := &cobra.Command{
		Use:   "colors",
		Short: "Print colored text to verify color support",
		Long: `Print colored text to verify color support.

The named colors are rendered with cmdio helpers, which check stdout for color
support regardless of --stream. --lipgloss adds truecolor and 256-color swatches
rendered through cmdio.NewRenderer for the chosen stream, so depth differences
are visible.`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch stream {
			case "", "stdout", "stderr":
				return nil
			default:
				return fmt.Errorf("--stream must be stdout or stderr, got %q", stream)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmd.ErrOrStderr()
			if stream == "stdout" {
				w = cmd.OutOrStdout()
			}

			swatch := "the quick brown fox jumps over the lazy dog"
			writeLine(ctx, stream, w, "red:     "+cmdio.Red(ctx, swatch))
			writeLine(ctx, stream, w, "green:   "+cmdio.Green(ctx, swatch))
			writeLine(ctx, stream, w, "yellow:  "+cmdio.Yellow(ctx, swatch))
			writeLine(ctx, stream, w, "blue:    "+cmdio.Blue(ctx, swatch))
			writeLine(ctx, stream, w, "cyan:    "+cmdio.Cyan(ctx, swatch))
			writeLine(ctx, stream, w, "hiblack: "+cmdio.HiBlack(ctx, swatch))
			writeLine(ctx, stream, w, "hiblue:  "+cmdio.HiBlue(ctx, swatch))

			if !swatches {
				return nil
			}
			r, _ := cmdio.NewRenderer(ctx, w)
			truecolor := make([]string, 0, swatchCount)
			ansi256 := make([]string, 0, swatchCount)
			for i := range swatchCount {
				v := i * 255 / (swatchCount - 1)
				hex := fmt.Sprintf("#%02x%02x%02x", v, 0, 255-v)
				truecolor = append(truecolor, r.NewStyle().Background(lipgloss.Color(hex)).Render("  "))
				index := 16 + i*7
				ansi256 = append(ansi256, r.NewStyle().Background(lipgloss.Color(strconv.Itoa(index))).Render("  "))
			}
			// lipgloss v2 styles always render full color; the renderer's
			// writer downsamples them to the stream's profile.
			sw := r.Writer(w)
			writeLine(ctx, stream, sw, "truecolor: "+strings.Join(truecolor, ""))
			writeLine(ctx, stream, sw, "256-color: "+strings.Join(ansi256, ""))
			return nil
		},
	}
	cmd.Flags().StringVar(&stream, "stream", "", "write output to stdout or stderr (default: stderr)")
	cmd.Flags().BoolVar(&swatches, "lipgloss", false, "also print truecolor and 256-color swatches")
	return cmd
}

// writeLine sends lines without --stream through cmdio.LogString, so the default
// output and its stderr routing are unchanged.
func writeLine(ctx context.Context, stream string, w io.Writer, s string) {
	if stream == "" {
		cmdio.LogString(ctx, s)
		return
	}
	fmt.Fprintln(w, s)
}
