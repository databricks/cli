package cmdiotest_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
)

// TestSelectBaseline_EscKey pins the current promptui-driven Select behavior
// when the user presses Esc at various states: the initial prompt, and after
// typing into the search filter. cmdio.Select uses StartInSearchMode: true,
// so the filter is active from the start.
//
// This test exists so the upcoming bubbletea replacement can be checked
// against a known-good baseline.
func TestSelectBaseline_EscKey(t *testing.T) {
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "g"},
	}, "Pick one")
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyEsc)
	tm.Golden("02-esc-from-initial")

	tm.Type("a")
	tm.Golden("03-after-typing-a")

	tm.Type(termtest.KeyEsc)
	tm.Golden("04-esc-clears-filter-or-not")

	type result struct {
		id  string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		id, err := tm.Result()
		resCh <- result{id: id, err: err}
	}()

	select {
	case res := <-resCh:
		t.Logf("prompt returned after Esc: id=%q err=%v", res.id, res.err)
		t.Logf("snapshot:\n%s", tm.Snapshot())
	case <-time.After(200 * time.Millisecond):
		tm.Type(termtest.KeyEnter)
		res := <-resCh
		t.Logf("prompt finalized with Enter: id=%q err=%v", res.id, res.err)
		t.Logf("snapshot:\n%s", tm.Snapshot())
	}
}
