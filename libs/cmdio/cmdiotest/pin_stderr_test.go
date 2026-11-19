package cmdiotest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/require"
)

// TestStderrPin_LogStringStyles pins the bytes cmdio.LogString writes to
// stderr for strings styled by the color helpers (which check stdout) and by
// a renderer bound to stderr. The cases vary which streams are terminals and
// whether NO_COLOR is set. The renderer's color is always plain here, because
// the screen is a pipe, so only the helper's styling varies in the goldens.
func TestStderrPin_LogStringStyles(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		opts termtest.Options
	}{
		{"stdout-tty-stderr-redirected", termtest.Options{StderrNotTTY: true}},
		{"stdout-redirected-stderr-tty", termtest.Options{StdoutNotTTY: true}},
		// NO_COLOR with both streams on a terminal: the helpers and the
		// renderer bound to stderr are plain.
		{"no-color-both-tty", termtest.Options{Env: colorEnvNamed("no-color"), ColorFromEnv: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tm := termtest.RunWith(t, tc.opts, func(ctx context.Context) (struct{}, error) {
				_, stderr := termtest.Streams(ctx)
				r, color := cmdio.NewRenderer(ctx, stderr)
				cmdio.LogString(ctx, fmt.Sprintf("renderer color: %t", color))
				cmdio.LogString(ctx, "helper: "+cmdio.Red(ctx, "red"))
				cmdio.LogString(ctx, "renderer: "+r.NewStyle().Bold(true).Render("bold"))
				return struct{}{}, nil
			})
			_, err := tm.Result()
			require.NoError(t, err)
			termtest.GoldenText(t, "01-stderr-bytes", quoted([]byte(tm.Raw())))
		})
	}
}
