package cmdiotest_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
)

// The modes pins record which terminal modes each component switches
// (cursor visibility, bracketed paste, keyboard protocols, ...) and which are
// still on when it returns. A mode left on after exit changes how the
// user's shell behaves.

// TestModesPin_Prompt pins the modes of a prompt on submit and on Ctrl+C.
func TestModesPin_Prompt(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, key string }{
		{"enter", termtest.KeyEnter},
		{"ctrl-c", termtest.KeyCtrlC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tm := newPinPrompt(t)
			typeKeys(tm, "ab")
			tm.Type(tc.key)
			_, _ = tm.Result()
			tm.GoldenModes("01-modes")
		})
	}
}

// TestModesPin_Select pins the modes of a select on submit and on Ctrl+C.
func TestModesPin_Select(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, key string }{
		{"enter", termtest.KeyEnter},
		{"ctrl-c", termtest.KeyCtrlC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tm := termtest.NewSelectOrdered(t, pinItems(), "Pick one")
			tm.WaitFor("gamma")
			// The select's Init and bubbletea's renderer each write a cursor
			// hide. A fast key can quit before the second one, which changes the
			// mode log, so wait for both. This wait is deliberately tolerant:
			// a version that hides once (the Charm v2 swap) must still reach
			// its golden, so the diff shows the change instead of a timeout.
			waitForRawCountOrGrace(t, tm, "\x1b[?25l", 2, 2*time.Second)
			tm.Type(tc.key)
			_, _ = tm.Result()
			tm.GoldenModes("01-modes")
		})
	}
}

// TestModesPin_Spinner pins the modes of a spinner that is closed normally.
// A spinner has no input, so there is no Ctrl+C case.
func TestModesPin_Spinner(t *testing.T) {
	t.Parallel()
	stop := make(chan struct{})
	stopOnce := sync.OnceFunc(func() { close(stop) })
	t.Cleanup(stopOnce)
	tm := termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Working")
		<-stop
		sp.Close()
		return struct{}{}, nil
	})
	tm.WaitFor("Working")
	stopOnce()
	_, _ = tm.Result()
	tm.GoldenModes("01-modes")
}

// TestModesPin_Pager pins the modes of the pager when it drains the list
// and when the user quits at the prompt between pages.
func TestModesPin_Pager(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, key string }{
		{"enter-drains", termtest.KeyEnter},
		{"ctrl-c", termtest.KeyCtrlC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			it := newPinIterator(120, -1, 0)
			tm := runPinPager(t, it)
			loadPage(tm, it)
			tm.WaitForScreen(pagerPrompt)
			tm.Settle()
			tm.Type(tc.key)
			_, _ = tm.Result()
			tm.GoldenModes("01-modes")
		})
	}
}

// waitForRawCountOrGrace waits until the program has written want copies of seq, or
// grace elapses. It never fails: the caller decides what a missing sequence
// means.
func waitForRawCountOrGrace(t *testing.T, tm *termtest.Term[string], seq string, want int, grace time.Duration) {
	t.Helper()
	deadline := time.Now().Add(grace)
	for strings.Count(tm.Raw(), seq) < want && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := strings.Count(tm.Raw(), seq); got < want {
		t.Logf("waited %s for %d×%q, saw %d; a missing mode line below may come from this", grace, want, seq, got)
	}
}
