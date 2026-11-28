package cmdiotest_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/require"
)

// TestColorPin_Swatch pins, for each environment of the color matrix, the
// color helpers and a renderer-minted style, and whether the renderer reports
// color for the screen. The color capability comes from the environment, as in
// a real command.
//
// Two limits. The screen is a pipe, so the renderer always reports no color
// here; a real-terminal renderer (depth, profile names) is not pinned. And
// cmdio reads only NO_COLOR and TERM=dumb, so the environments other than
// no-color and term-dumb produce identical goldens. That is deliberate: it
// fails if cmdio starts reading a variable that a row sets (TERM, CI,
// CLICOLOR, COLORTERM or NO_COLOR).
func TestColorPin_Swatch(t *testing.T) {
	t.Parallel()
	for _, ce := range colorEnvs() {
		t.Run(ce.name, func(t *testing.T) {
			t.Parallel()
			tm := termtest.RunWith(t, termtest.Options{Env: ce.env, ColorFromEnv: true, MapNewline: true}, func(ctx context.Context) (struct{}, error) {
				_, screen := termtest.Streams(ctx)
				r, color := cmdio.NewRenderer(ctx, screen)
				cmdio.LogString(ctx, fmt.Sprintf("renderer color: %t", color))
				cmdio.LogString(ctx, "helpers: "+cmdio.Red(ctx, "red")+" "+cmdio.Green(ctx, "green")+" "+cmdio.Blue(ctx, "blue")+" "+
					cmdio.HiBlack(ctx, "hiblack")+" "+cmdio.HiBlue(ctx, "hiblue")+" "+cmdio.Bold(ctx, "bold")+" "+cmdio.Faint(ctx, "faint"))
				cmdio.LogString(ctx, "renderer: "+r.NewStyle().Bold(true).Render("bold")+" "+r.NewStyle().Reverse(true).Render("reverse"))
				return struct{}{}, nil
			})
			_, err := tm.Result()
			require.NoError(t, err)
			tm.GoldenStyled("01-swatch")
		})
	}
}

// TestColorPin_Select pins the styled screen of a select prompt for each
// environment of the color matrix.
func TestColorPin_Select(t *testing.T) {
	t.Parallel()
	for _, ce := range colorEnvs() {
		t.Run(ce.name, func(t *testing.T) {
			t.Parallel()
			tm := termtest.RunWith(t, termtest.Options{Env: ce.env, ColorFromEnv: true}, func(ctx context.Context) (string, error) {
				return cmdio.SelectOrdered(ctx, pinItems(), "Pick an item")
			})
			if !tm.WaitForOrExit("gamma") {
				_, err := tm.Result()
				termtest.GoldenText(t, "01-not-shown", fmt.Sprintf("err: %v", err))
				return
			}
			tm.GoldenStyled("01-initial")
			tm.Type(termtest.KeyEnter)
			_, err := tm.Result()
			require.NoError(t, err)
		})
	}
}

// TestColorPin_Prompt pins the styled screen of a text prompt for each
// environment of the color matrix.
func TestColorPin_Prompt(t *testing.T) {
	t.Parallel()
	for _, ce := range colorEnvs() {
		t.Run(ce.name, func(t *testing.T) {
			t.Parallel()
			tm := termtest.RunWith(t, termtest.Options{Env: ce.env, ColorFromEnv: true}, func(ctx context.Context) (string, error) {
				return cmdio.RunPrompt(ctx, cmdio.PromptOptions{Label: "Workspace name"})
			})
			if !tm.WaitForOrExit("Workspace name") {
				_, err := tm.Result()
				termtest.GoldenText(t, "01-not-shown", fmt.Sprintf("err: %v", err))
				return
			}
			typeKeys(tm, "dev")
			tm.GoldenStyled("01-typed")
			tm.Type(termtest.KeyEnter)
			_, err := tm.Result()
			require.NoError(t, err)
		})
	}
}

// TestColorPin_Spinner pins the styled screen of a running spinner for each
// environment of the color matrix. Only TERM=dumb gets no spinner; NO_COLOR
// keeps it, uncolored.
func TestColorPin_Spinner(t *testing.T) {
	t.Parallel()
	for _, ce := range colorEnvs() {
		t.Run(ce.name, func(t *testing.T) {
			t.Parallel()
			stop := make(chan struct{})
			stopOnce := sync.OnceFunc(func() { close(stop) })
			t.Cleanup(stopOnce)
			tm := termtest.RunWith(t, termtest.Options{Env: ce.env, ColorFromEnv: true, MapNewline: true, Normalize: blankSpinnerFrames}, func(ctx context.Context) (struct{}, error) {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("Working")
				cmdio.LogString(ctx, "spinner started")
				<-stop
				sp.Close()
				return struct{}{}, nil
			})
			tm.WaitFor("spinner started")
			tm.GoldenStyled("01-running")
			stopOnce()
			_, err := tm.Result()
			require.NoError(t, err)
		})
	}
}
