package cmdiotest_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
)

// pagerPrompt is the text the pager shows between pages.
const pagerPrompt = "[space] more"

// pinRow is the element type the pager pins list.
type pinRow struct {
	Name  string
	State string
}

// pinIterator is a listing.Iterator over n rows that stops before each row
// index in holds until the test calls release, so a test can look at the
// pager while a page is loading. With failAt >= 0 it returns an error
// instead of row failAt.
type pinIterator struct {
	n, failAt int
	holds     []int
	gates     []chan struct{}

	mu       sync.Mutex
	i        int
	released int
}

// pinPageSize is the pager's page size in tests: without a window size it
// uses its fallback of 50 rows. Only that fallback is pinned. RenderIterator
// starts its own program and gets no window size, so the path that sizes the
// page from the terminal height is not reached by these pins.
const pinPageSize = 50

func newPinIterator(n, failAt int, holds ...int) *pinIterator {
	it := &pinIterator{n: n, failAt: failAt, holds: holds}
	for range holds {
		it.gates = append(it.gates, make(chan struct{}))
	}
	return it
}

// release lets the iterator past its next hold.
func (it *pinIterator) release() {
	it.mu.Lock()
	defer it.mu.Unlock()
	close(it.gates[it.released])
	it.released++
}

func (it *pinIterator) HasNext(ctx context.Context) bool {
	it.mu.Lock()
	i := it.i
	it.mu.Unlock()
	for h, at := range it.holds {
		if at == i {
			select {
			case <-it.gates[h]:
			case <-ctx.Done():
				return false
			}
		}
	}
	it.mu.Lock()
	defer it.mu.Unlock()
	return it.i < it.n || it.i == it.failAt
}

func (it *pinIterator) Next(ctx context.Context) (pinRow, error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	if it.i == it.failAt {
		return pinRow{}, errors.New("injected iterator error")
	}
	it.i++
	return pinRow{Name: fmt.Sprintf("item-%03d", it.i), State: "ready"}, nil
}

// loadPage waits until the pager shows its loading spinner and then lets the
// iterator past its next hold, so every page lands after the spinner has
// been painted. Whether it was painted decides which characters a printed
// row leaves behind, so this keeps the screens deterministic.
func loadPage(tm *termtest.Term[struct{}], it *pinIterator) {
	tm.WaitForScreen("loading...")
	it.release()
}

// runPinPager renders it through cmdio.RenderIterator with a row template,
// so the production pager gate pages it. Screens keep only the printed rows
// on their lines; see onlyPagerRows.
func runPinPager(t *testing.T, it *pinIterator) *termtest.Term[struct{}] {
	t.Helper()
	tm := termtest.RunWith(t, termtest.Options{
		CaptureStdout:  true,
		HeaderTemplate: "NAME\tSTATE",
		Template:       "{{range .}}{{.Name}}\t{{.State}}\n{{end}}",
		ScreenRows:     200,
		Normalize:      onlyPagerRows,
	}, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, cmdio.RenderIterator(ctx, it)
	})
	tm.LogRawOnFailure()
	return tm
}

// pagerRowLine matches a header or data row the pager printed, ignoring
// what follows it on the same screen line.
var pagerRowLine = regexp.MustCompile(`(?m)^(NAME +STATE|item-\d{3} +ready).*$`)

// onlyPagerRows keeps just the printed row on each line that starts with one.
// Part of the spinner or prompt line may or may not survive next to a
// printed row, depending on whether the renderer painted its view before or
// after the rows were flushed. That leftover is timing, not behavior, so the
// pager pins drop it.
func onlyPagerRows(s string) string {
	return pagerRowLine.ReplaceAllString(s, "$1")
}

func pinPagerResult(t *testing.T, tm *termtest.Term[struct{}], step string) {
	t.Helper()
	_, err := tm.Result()
	termtest.GoldenText(t, step, fmt.Sprintf("err: %v", err))
}

// observeSpinnerFrames waits up to d for want distinct spinner frames and
// describes what it saw. Falling short is not a test failure: the count is
// the behaviour being pinned, so a spinner that stops animating shows up as a
// golden diff. The wait is bounded by d.
func observeSpinnerFrames(tm *termtest.Term[struct{}], want int, d time.Duration) string {
	deadline := time.Now().Add(d)
	for {
		n := countSpinnerFrames(tm.Raw())
		if n >= want {
			return fmt.Sprintf("distinct spinner frames: at least %d", want)
		}
		if time.Now().After(deadline) {
			return fmt.Sprintf("distinct spinner frames: %d (fewer than %d within %s)", n, want, d)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestPagerPin_SpinnerDuringFirstPage pins whether the spinner animates
// while the first page loads, and the screen once it has landed.
func TestPagerPin_SpinnerDuringFirstPage(t *testing.T) {
	t.Parallel()
	it := newPinIterator(120, -1, 0)
	tm := runPinPager(t, it)
	termtest.GoldenText(t, "01-first-page-loading", observeSpinnerFrames(tm, 3, 3*time.Second))
	loadPage(tm, it)
	tm.WaitForScreen(pagerPrompt)
	tm.Golden("02-first-page")
	tm.Type("q")
	pinPagerResult(t, tm, "03-result")
}

// TestPagerPin_EnterDrainsRemainingPages pins Enter pressed while the first
// page loads: whether the spinner animates and what ends up on screen once
// everything has been drained. The key may reach the pager during the fetch
// or at the prompt after it; nothing observable tells which, and both drain
// the list to the same final screen.
func TestPagerPin_EnterDrainsRemainingPages(t *testing.T) {
	t.Parallel()
	it := newPinIterator(120, -1, 0)
	tm := runPinPager(t, it)
	termtest.GoldenText(t, "01-first-page-loading", observeSpinnerFrames(tm, 2, 3*time.Second))
	tm.Type(termtest.KeyEnter)
	loadPage(tm, it)
	pinPagerResult(t, tm, "02-result")
	tm.Golden("03-final")
}

// TestPagerPin_SpaceLoadsNextPage pins one more page on space.
//
// Settle keeps space from arriving while a render of the first page is still
// in flight. On failure the test logs the raw output stream for diagnosis.
func TestPagerPin_SpaceLoadsNextPage(t *testing.T) {
	t.Parallel()
	it := newPinIterator(120, -1, 0, pinPageSize)
	tm := runPinPager(t, it)
	loadPage(tm, it)
	tm.WaitForScreen(pagerPrompt)
	tm.Settle()
	tm.Type(" ")
	loadPage(tm, it)
	tm.WaitForScreen("item-100")
	tm.WaitForScreen(pagerPrompt)
	tm.Golden("01-second-page")
	tm.Type("q")
	pinPagerResult(t, tm, "02-result")
}

// TestPagerPin_ErrorOnFirstPage pins an iterator error before any row.
func TestPagerPin_ErrorOnFirstPage(t *testing.T) {
	t.Parallel()
	it := newPinIterator(120, 0, 0)
	tm := runPinPager(t, it)
	loadPage(tm, it)
	pinPagerResult(t, tm, "01-result")
	tm.Golden("02-final")
}

// TestPagerPin_ErrorOnLaterPage pins an iterator error on the second page.
func TestPagerPin_ErrorOnLaterPage(t *testing.T) {
	t.Parallel()
	it := newPinIterator(120, 60, 0, pinPageSize)
	tm := runPinPager(t, it)
	loadPage(tm, it)
	tm.WaitForScreen(pagerPrompt)
	tm.Settle()
	tm.Type(" ")
	loadPage(tm, it)
	pinPagerResult(t, tm, "01-result")
	tm.Golden("02-final")
}

// TestPagerPin_EmptyResult pins a list with no rows.
func TestPagerPin_EmptyResult(t *testing.T) {
	t.Parallel()
	it := newPinIterator(0, -1, 0)
	tm := runPinPager(t, it)
	loadPage(tm, it)
	pinPagerResult(t, tm, "01-result")
	tm.Golden("02-final")
}

// TestPagerPin_FitsOnePage pins a list shorter than a page.
func TestPagerPin_FitsOnePage(t *testing.T) {
	t.Parallel()
	it := newPinIterator(5, -1, 0)
	tm := runPinPager(t, it)
	loadPage(tm, it)
	pinPagerResult(t, tm, "01-result")
	tm.Golden("02-final")
}
