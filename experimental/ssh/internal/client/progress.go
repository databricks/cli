package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/cmdio"
)

// progressUI renders the bootstrap steps: each step spins while it runs and
// leaves a checkmark line when it finishes. Subprocess output is captured per
// step and printed only when that step fails.
type progressUI struct {
	ctx   context.Context
	w     io.Writer
	check string // styled "✓" prefix for a finished step
	cross string // styled "✗" prefix for a failed step
}

// newProgressUI builds a progress renderer writing its checkmark lines to stderr.
func newProgressUI(ctx context.Context) *progressUI {
	// Mint the checkmark styles from a renderer targeting stderr so color handling
	// stays centralized in cmdio.NewRenderer, matching cmdio's own spinner.
	r, _ := cmdio.NewRenderer(ctx, os.Stderr)
	return &progressUI{
		ctx:   ctx,
		w:     os.Stderr,
		check: r.NewStyle().Foreground(lipgloss.Color("10")).Render("✓"), // green
		cross: r.NewStyle().Foreground(lipgloss.Color("9")).Render("✗"),  // red
	}
}

// runStep shows a cmdio spinner labelled desc while fn runs, giving fn a writer
// that captures the step's subprocess output. On success it leaves a checkmark
// line; on failure it prints the captured output (stdout+stderr, in order) before
// returning fn's error. The shared spinner degrades to no output in a
// non-interactive terminal, so the checkmark line is what the reader sees either
// way.
func (ui *progressUI) runStep(desc string, fn func(out io.Writer) error) error {
	sp := cmdio.NewSpinner(ui.ctx)
	sp.Update(desc)

	var buf bytes.Buffer
	err := fn(&buf)
	// Stop the spinner (clearing its line) before printing the step's outcome.
	sp.Close()

	if err != nil {
		fmt.Fprintln(ui.w, ui.cross+" "+desc)
		if out := buf.String(); out != "" {
			fmt.Fprint(ui.w, out)
			if !strings.HasSuffix(out, "\n") {
				fmt.Fprintln(ui.w)
			}
		}
		return err
	}

	fmt.Fprintln(ui.w, ui.check+" "+desc)
	return nil
}
