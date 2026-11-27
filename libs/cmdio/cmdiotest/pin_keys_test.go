package cmdiotest_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
)

// pinPromptResult pins a prompt's outcome: the value, or the error.
func pinPromptResult(t *testing.T, tm *termtest.Term[string], step string) {
	t.Helper()
	v, err := tm.Result()
	termtest.GoldenText(t, step, fmt.Sprintf("value: %q\nerr: %v", v, err))
}

func newPinPrompt(t *testing.T) *termtest.Term[string] {
	t.Helper()
	tm := termtest.NewPrompt(t, cmdio.PromptOptions{Label: "Value"})
	tm.WaitFor("Value")
	return tm
}

// TestKeysPin_PromptBurstKeyName pins what a prompt does with text that is
// also a key name ("enter") when it arrives in one read, as it does when
// typed fast over a slow link.
func TestKeysPin_PromptBurstKeyName(t *testing.T) {
	t.Parallel()
	tm := newPinPrompt(t)
	typeKeys(tm, "a")
	tm.TypeBurst("enter")
	tm.Settle()
	typeKeys(tm, "b")
	tm.Golden("01-after-burst")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "02-result")
}

// TestKeysPin_SelectFilterBurstKeyName pins what select's search does with a
// burst of "down", a key name, typed as text.
func TestKeysPin_SelectFilterBurstKeyName(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "edown", Id: "id-01"},
		{Name: "elephant", Id: "id-02"},
		{Name: "eagle", Id: "id-03"},
	}, "Pick one")
	tm.WaitFor("eagle")
	typeKeys(tm, "e")
	tm.TypeBurst("down")
	tm.Settle()
	tm.Golden("01-after-burst")
	tm.Type(termtest.KeyEnter)
	id, err := tm.Result()
	termtest.GoldenText(t, "02-result", fmt.Sprintf("id: %q\nerr: %v", id, err))
}

// keyExitGrace is how long a test waits for a key that may quit the prompt,
// before typing more. Typing sooner races the quit: the next key can be queued
// ahead of the quit and land in the value. The wait returns as soon as the
// prompt exits, so the grace is only spent when the key is ignored or the exit
// is slow. A slow exit within the grace is not a failure of its own; the
// golden after the wait then shows it.
const keyExitGrace = 5 * time.Second

// TestKeysPin_PromptAltEnter pins Alt+Enter (Esc CR) in a prompt.
func TestKeysPin_PromptAltEnter(t *testing.T) {
	t.Parallel()
	tm := newPinPrompt(t)
	typeKeys(tm, "ab", termtest.KeyAltEnter)
	tm.WaitForExit(keyExitGrace)
	typeKeys(tm, "c")
	tm.Golden("01-after-alt-enter")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "02-result")
}

// TestKeysPin_SelectAltEnter pins Alt+Enter in a select.
func TestKeysPin_SelectAltEnter(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, pinItems(), "Pick one")
	tm.WaitFor("gamma")
	typeKeys(tm, termtest.KeyDown, termtest.KeyAltEnter)
	tm.Golden("01-after-alt-enter")
	tm.Type(termtest.KeyEnter)
	id, err := tm.Result()
	termtest.GoldenText(t, "02-result", fmt.Sprintf("id: %q\nerr: %v", id, err))
}

// TestKeysPin_PromptAltLeft pins Alt+Left (CSI 1;3D) in the middle of a word.
func TestKeysPin_PromptAltLeft(t *testing.T) {
	t.Parallel()
	tm := newPinPrompt(t)
	typeKeys(tm, "abc", termtest.KeyAltLeft, "X")
	tm.Golden("01-after-alt-left")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "02-result")
}

// TestKeysPin_PromptShiftEnter pins Shift+Enter in the two encodings
// terminals use once the program enables an extended keyboard protocol.
func TestKeysPin_PromptShiftEnter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, key string }{
		{"kitty", termtest.KeyShiftEnterKitty},
		{"modifyOtherKeys", termtest.KeyShiftEnterModifyOtherKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tm := newPinPrompt(t)
			typeKeys(tm, "xy", tc.key)
			tm.WaitForExit(keyExitGrace)
			typeKeys(tm, "z")
			tm.Golden("01-after-shift-enter")
			tm.Type(termtest.KeyEnter)
			pinPromptResult(t, tm, "02-result")
		})
	}
}

// TestKeysPin_SelectShiftEnter pins Shift+Enter in a select.
func TestKeysPin_SelectShiftEnter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, key string }{
		{"kitty", termtest.KeyShiftEnterKitty},
		{"modifyOtherKeys", termtest.KeyShiftEnterModifyOtherKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tm := termtest.NewSelectOrdered(t, pinItems(), "Pick one")
			tm.WaitFor("gamma")
			typeKeys(tm, termtest.KeyDown, tc.key)
			tm.Golden("01-after-shift-enter")
			tm.Type(termtest.KeyEnter)
			id, err := tm.Result()
			termtest.GoldenText(t, "02-result", fmt.Sprintf("id: %q\nerr: %v", id, err))
		})
	}
}

// TestKeysPin_PromptKittyEsc pins Esc in its kitty keyboard encoding.
func TestKeysPin_PromptKittyEsc(t *testing.T) {
	t.Parallel()
	tm := newPinPrompt(t)
	typeKeys(tm, "ab", termtest.KeyEscKitty)
	tm.Golden("01-after-kitty-esc")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "02-result")
}

// TestKeysPin_PromptBracketedPasteWithNewline pins a bracketed paste that
// contains a newline.
func TestKeysPin_PromptBracketedPasteWithNewline(t *testing.T) {
	t.Parallel()
	tm := newPinPrompt(t)
	typeKeys(tm, termtest.Paste("a\r\nb"))
	tm.Golden("01-after-paste")
	tm.Type(termtest.KeyEnter)
	pinPromptResult(t, tm, "02-result")
}
