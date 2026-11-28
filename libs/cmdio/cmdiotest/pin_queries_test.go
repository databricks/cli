package cmdiotest_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
)

// slowTerminal answers every query, but only after the program has had
// time to quit, the way a terminal over a slow link does.
var slowTerminal = &termtest.Responder{Delay: 300 * time.Millisecond}

func skipLeftoverOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("on Windows bubbletea's input reader can outlive the program and take the leftover bytes first")
	}
}

// TestQueriesPin_PromptQuitBeforeReplies pins the terminal queries a prompt
// writes and the reply bytes left unread when it quits before a slow
// terminal answers. Those bytes are what the shell reads next.
func TestQueriesPin_PromptQuitBeforeReplies(t *testing.T) {
	t.Parallel()
	skipLeftoverOnWindows(t)
	tm := termtest.RunWith(t, termtest.Options{Responder: slowTerminal, CaptureLeftover: true}, func(ctx context.Context) (string, error) {
		return cmdio.RunPrompt(ctx, cmdio.PromptOptions{Label: "Value"})
	})
	tm.WaitFor("Value")
	typeKeys(tm, "ab")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "01-result")
	tm.GoldenQueries("02-queries")
	leftover, err := tm.Leftover()
	if err != nil {
		t.Fatal(err)
	}
	termtest.GoldenText(t, "03-leftover", quoted(leftover))
}

// TestQueriesPin_SelectQuitBeforeReplies is the select counterpart of
// TestQueriesPin_PromptQuitBeforeReplies.
func TestQueriesPin_SelectQuitBeforeReplies(t *testing.T) {
	t.Parallel()
	skipLeftoverOnWindows(t)
	tm := termtest.RunWith(t, termtest.Options{Responder: slowTerminal, CaptureLeftover: true}, func(ctx context.Context) (string, error) {
		return cmdio.SelectOrdered(ctx, pinItems(), "Pick one")
	})
	tm.WaitFor("gamma")
	tm.Type(termtest.KeyEnter)
	_, _ = tm.Result()
	tm.GoldenQueries("01-queries")
	leftover, err := tm.Leftover()
	if err != nil {
		t.Fatal(err)
	}
	termtest.GoldenText(t, "02-leftover", quoted(leftover))
}

// TestQueriesPin_PromptTerminalAnswersAtOnce pins the queries and leftover
// when the terminal answers at once, before the user quits.
func TestQueriesPin_PromptTerminalAnswersAtOnce(t *testing.T) {
	t.Parallel()
	skipLeftoverOnWindows(t)
	tm := termtest.RunWith(t, termtest.Options{Responder: &termtest.Responder{}, CaptureLeftover: true}, func(ctx context.Context) (string, error) {
		return cmdio.RunPrompt(ctx, cmdio.PromptOptions{Label: "Value"})
	})
	tm.WaitFor("Value")
	// Replies are written on a poll, so without this they could land between
	// the typed keys. The golden is the same either way, but the wait keeps
	// the reply and the key from racing.
	tm.WaitForReplies()
	typeKeys(tm, "ab")
	tm.Golden("01-typed")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "02-result")
	tm.GoldenQueries("03-queries")
	leftover, err := tm.Leftover()
	if err != nil {
		t.Fatal(err)
	}
	termtest.GoldenText(t, "04-leftover", quoted(leftover))
}

// TestQueriesPin_PromptUnsolicitedBackgroundReply pins what a prompt does
// with an OSC 11 (background color) reply that arrives on its input without
// it having asked, e.g. a late answer to a query written before the prompt
// started.
func TestQueriesPin_PromptUnsolicitedBackgroundReply(t *testing.T) {
	t.Parallel()
	tm := newPinPrompt(t)
	typeKeys(tm, "ab", termtest.DefaultReply("OSC 11"))
	tm.Golden("01-after-reply")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "02-result")
}
