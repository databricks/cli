package tui

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
			// Default keeps the historical cmdio.LogString path, which writes to stderr.
			var line func(string)
			var w io.Writer
			switch stream {
			case "stdout":
				w = cmd.OutOrStdout()
			default:
				w = cmd.ErrOrStderr()
			}
			if stream == "" {
				line = func(s string) { cmdio.LogString(ctx, s) }
			} else {
				line = func(s string) { fmt.Fprintln(w, s) }
			}

			swatch := "the quick brown fox jumps over the lazy dog"
			line("red:     " + cmdio.Red(ctx, swatch))
			line("green:   " + cmdio.Green(ctx, swatch))
			line("yellow:  " + cmdio.Yellow(ctx, swatch))
			line("blue:    " + cmdio.Blue(ctx, swatch))
			line("cyan:    " + cmdio.Cyan(ctx, swatch))
			line("hiblack: " + cmdio.HiBlack(ctx, swatch))
			line("hiblue:  " + cmdio.HiBlue(ctx, swatch))

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
			line("truecolor: " + strings.Join(truecolor, ""))
			line("256-color: " + strings.Join(ansi256, ""))
			return nil
		},
	}
	cmd.Flags().StringVar(&stream, "stream", "", "write output to stdout or stderr (default: cmdio.LogString, stderr)")
	cmd.Flags().BoolVar(&swatches, "lipgloss", false, "also print truecolor and 256-color swatches")
	return cmd
}
