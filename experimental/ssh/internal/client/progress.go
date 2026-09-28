package client

import (
	"bytes"
	"context"
	"io"

	"github.com/databricks/cli/libs/cmdio"
)

// runStep shows a cmdio spinner labelled desc while fn runs, giving fn a writer
// that captures the step's subprocess output. On success it leaves a checkmark
// line; on failure it prints the captured output (stdout+stderr, in order) before
// returning fn's error. The shared spinner shows elapsed time and degrades to no
// output in a non-interactive terminal, so the checkmark line is what the reader
// sees either way.
func runStep(ctx context.Context, desc string, fn func(out io.Writer) error) error {
	sp := cmdio.NewSpinner(ctx, cmdio.WithElapsedTime())
	// Close is idempotent; defer it so a panic in fn can't leave the spinner (and
	// its tea-program slot) running, while the explicit Close below still controls
	// output ordering on the normal path.
	defer sp.Close()
	sp.Update(desc)

	var buf bytes.Buffer
	err := fn(&buf)
	// Stop the spinner (clearing its line) before printing the step's outcome.
	sp.Close()

	if err != nil {
		cmdio.LogProgress(ctx, cmdio.Red(ctx, "✗ "+desc))
		if out := buf.String(); out != "" {
			cmdio.LogString(ctx, out)
		}
		return err
	}

	cmdio.LogProgress(ctx, cmdio.Green(ctx, "✓ "+desc))
	return nil
}
