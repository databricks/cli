package cmdio

import (
	"errors"
	"io"
	"reflect"
	"testing"
	"text/template"

	tea "charm.land/bubbletea/v2"
	"github.com/databricks/cli/libs/cmdio/teainput"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPager(t *testing.T, iter listing.Iterator[int], pageSize int) *pagerModel[int] {
	t.Helper()
	ctx := MockDiscard(t.Context())
	fm := renderFuncMap(ctx)
	rowT, err := template.New("row").Funcs(fm).Parse("{{range .}}{{.}}\n{{end}}")
	require.NoError(t, err)
	headerT, err := template.New("header").Funcs(fm).Parse("")
	require.NoError(t, err)
	return newPagerModel(ctx, io.Discard, iter, &templatePager{
		headerT: headerT,
		rowT:    rowT,
	}, pageSize, 0)
}

func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	require.NotNil(t, cmd)
	return cmd()
}

// unwrapCmds pulls the cmds out of a tea.Batch/tea.Sequence result.
// sequenceMsg is unexported, so we fall back to reflect on the []tea.Cmd
// underlying type — update if bubbletea renames it.
func unwrapCmds(t *testing.T, msg tea.Msg) []tea.Cmd {
	t.Helper()
	if bm, ok := msg.(tea.BatchMsg); ok {
		return []tea.Cmd(bm)
	}
	rv := reflect.ValueOf(msg)
	require.Equal(t, reflect.Slice, rv.Kind(), "expected a slice-of-cmds msg, got %T", msg)
	cmds := make([]tea.Cmd, rv.Len())
	for i := range cmds {
		c, ok := reflect.TypeAssert[tea.Cmd](rv.Index(i))
		require.True(t, ok, "slice element %d is not a tea.Cmd", i)
		cmds[i] = c
	}
	return cmds
}

// printedText pulls the body out of a tea.Println result.
// printLineMessage is unexported, so reflect on its only string field.
func printedText(t *testing.T, msg tea.Msg) string {
	t.Helper()
	rv := reflect.ValueOf(msg)
	require.Equal(t, reflect.Struct, rv.Kind(), "expected a struct msg, got %T", msg)
	for _, field := range rv.Fields() {
		if field.Kind() == reflect.String {
			return field.String()
		}
	}
	t.Fatalf("no string field in %T", msg)
	return ""
}

func TestPagerModelInitFetchesFirstBatch(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 3}, 10)
	// Init returns a tea.Batch(m.fetch, spinner.Tick); find the fetch.
	var b batchMsg
	for _, c := range unwrapCmds(t, runCmd(t, m.Init())) {
		if msg, ok := c().(batchMsg); ok {
			b = msg
			break
		}
	}
	assert.True(t, b.done, "small iterator is drained in one batch")
	assert.Len(t, b.lines, 3)
	assert.True(t, m.fetching, "Init must mark the model as fetching")
}

func TestPagerModelDoFetchLimit(t *testing.T) {
	cases := []struct {
		name      string
		pageSize  int
		limit     int
		total     int
		wantCount int
		wantDone  bool
	}{
		{"exact boundary marks done", 5, 5, 0, 5, true},
		{"limit reached mid-page", 5, 12, 10, 2, true},
		{"page boundary below limit not done", 5, 100, 0, 5, false},
		{"no limit not done", 5, 0, 0, 5, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestPager(t, &numberIterator{n: 100}, tc.pageSize)
			msg := m.doFetch(t.Context(), tc.pageSize, tc.limit, tc.total)
			b, ok := msg.(batchMsg)
			require.True(t, ok, "expected batchMsg, got %T", msg)
			require.NoError(t, b.err)
			assert.Equal(t, tc.wantCount, b.count)
			assert.Equal(t, tc.wantDone, b.done)
		})
	}
}

func TestPagerModelBatchPrintsAndQuitsWhenDone(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 3}, 10)
	_, cmd := m.Update(batchMsg{lines: []string{"1", "2", "3"}, done: true})
	assert.True(t, m.iterDone)
	assert.True(t, m.hasPrinted)
	cmds := unwrapCmds(t, runCmd(t, cmd))
	require.Len(t, cmds, 2)
	assert.Contains(t, printedText(t, runCmd(t, cmds[0])), "1\n2\n3")
	_, isQuit := runCmd(t, cmds[1]).(tea.QuitMsg)
	assert.True(t, isQuit, "final cmd must quit once the iterator is drained")
}

func TestPagerModelBatchDonePrintsHeaderOnlyEmptyIter(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 0}, 10)
	_, cmd := m.Update(batchMsg{lines: []string{"HEADER"}, done: true})
	cmds := unwrapCmds(t, runCmd(t, cmd))
	require.Len(t, cmds, 2)
	assert.Equal(t, "HEADER", printedText(t, runCmd(t, cmds[0])))
}

func TestPagerModelBatchPromptsWhenMore(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	_, cmd := m.Update(batchMsg{lines: []string{"1", "2"}, done: false})
	assert.False(t, m.iterDone)
	assert.True(t, m.hasPrinted)
	assert.False(t, m.drainAll)
	assert.True(t, m.printing)
	assert.Equal(t, pagerPromptText, m.render())
	cmds := unwrapCmds(t, runCmd(t, cmd))
	require.Len(t, cmds, 2)
	assert.Contains(t, printedText(t, runCmd(t, cmds[0])), "1\n2")
	printed := runCmd(t, cmds[1])
	require.IsType(t, pagePrintedMsg{}, printed)

	_, cmd = m.Update(printed)
	assert.Nil(t, cmd)
	assert.False(t, m.printing)
	assert.Equal(t, pagerPromptText, m.render())
}

// TestPagerModelKeysWaitForPagePrint covers the window between a batchMsg
// and its pagePrintedMsg: tea runs each Cmd on its own goroutine, so a
// fetch or Quit issued then could overtake the page's print and drop rows.
func TestPagerModelKeysWaitForPagePrint(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	_, _ = m.Update(batchMsg{lines: []string{"1", "2"}, done: false})

	_, cmd := m.Update(teainput.Press("space"))
	assert.Nil(t, cmd, "SPACE before the page is printed must not fetch")
	_, cmd = m.Update(teainput.Press("enter"))
	assert.Nil(t, cmd, "ENTER before the page is printed must not fetch")
	assert.True(t, m.drainAll)

	_, cmd = m.Update(pagePrintedMsg{})
	_, isFetch := runCmd(t, cmd).(batchMsg)
	assert.True(t, isFetch, "the deferred drain fetches once the page is printed")
}

func TestPagerModelQuitWaitsForPagePrint(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	_, _ = m.Update(batchMsg{lines: []string{"1", "2"}, done: false})

	_, cmd := m.Update(teainput.Press("q"))
	assert.Nil(t, cmd, "q before the page is printed must not quit yet")

	_, cmd = m.Update(pagePrintedMsg{})
	_, isQuit := runCmd(t, cmd).(tea.QuitMsg)
	assert.True(t, isQuit, "the deferred quit fires once the page is printed")
}

func TestPagerModelBatchDrainingChainsFetch(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.drainAll = true
	m.fetching = true
	_, cmd := m.Update(batchMsg{lines: []string{"1", "2"}, done: false})
	cmds := unwrapCmds(t, runCmd(t, cmd))
	require.Len(t, cmds, 2)
	assert.Contains(t, printedText(t, runCmd(t, cmds[0])), "1\n2")
	_, cmd = m.Update(runCmd(t, cmds[1]))
	assert.True(t, m.fetching)
	_, isFetch := runCmd(t, cmd).(batchMsg)
	assert.True(t, isFetch, "draining must auto-fetch the next batch")
}

func TestPagerModelBatchErrorTerminates(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 0}, 5)
	_, cmd := m.Update(batchMsg{err: errors.New("boom")})
	assert.EqualError(t, m.err, "boom")
	_, isQuit := runCmd(t, cmd).(tea.QuitMsg)
	assert.True(t, isQuit)
}

func TestPagerModelSpaceFetchesNext(t *testing.T) {
	cases := []struct {
		name string
		key  tea.Msg
	}{
		{"KeySpace", teainput.Press("space")},
		{"KeyRunes space", teainput.Type(" ")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestPager(t, &numberIterator{n: 100}, 5)
			m.hasPrinted = true
			_, cmd := m.Update(tc.key)
			_, ok := runCmd(t, cmd).(batchMsg)
			assert.True(t, ok, "space should fire a fetch")
		})
	}
}

func TestPagerModelEnterSetsDrainAll(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.hasPrinted = true
	_, cmd := m.Update(teainput.Press("enter"))
	assert.True(t, m.drainAll)
	assert.NotContains(t, m.render(), pagerPromptText, "no prompt while draining")
	_, ok := runCmd(t, cmd).(batchMsg)
	assert.True(t, ok, "enter should fire a fetch")
}

func TestPagerModelEnterIsNoOpWhileAlreadyDraining(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.hasPrinted = true
	m.drainAll = true
	_, cmd := m.Update(teainput.Press("enter"))
	assert.Nil(t, cmd, "re-entering drain shouldn't stack another fetch")
}

func TestPagerModelSpaceIgnoredDuringFetch(t *testing.T) {
	// Between Init and the first batchMsg, SPACE must not schedule a second
	// fetch: doing so would run the iterator from two goroutines at once.
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	_, cmd := m.Update(teainput.Press("space"))
	assert.Nil(t, cmd, "SPACE while fetching must not dispatch another fetch")
}

func TestPagerModelEnterDuringFetchDefersFetch(t *testing.T) {
	// ENTER during an in-flight fetch flips drainAll but can't issue a new
	// fetch; the pagePrintedMsg after the pending batch chains one.
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	_, cmd := m.Update(teainput.Press("enter"))
	assert.True(t, m.drainAll)
	assert.Nil(t, cmd, "ENTER during fetch must defer to pagePrintedMsg chaining")
}

func TestPagerModelQuitKeys(t *testing.T) {
	cases := []struct {
		name string
		key  tea.Msg
	}{
		{"q", teainput.Press("q")},
		{"Q", teainput.Press("Q")},
		{"esc", teainput.Press("esc")},
		{"alt+esc", teainput.Press("alt+esc")},
		{"ctrl+c", teainput.Press("ctrl+c")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestPager(t, &numberIterator{n: 100}, 5)
			m.hasPrinted = true
			_, cmd := m.Update(tc.key)
			_, ok := runCmd(t, cmd).(tea.QuitMsg)
			assert.True(t, ok)
		})
	}
}

func TestPagerModelQuitKeysInterruptDrain(t *testing.T) {
	for _, key := range []tea.Msg{
		teainput.Press("q"),
		teainput.Press("esc"),
		teainput.Press("ctrl+c"),
	} {
		m := newTestPager(t, &numberIterator{n: 100}, 5)
		m.hasPrinted = true
		m.drainAll = true
		_, cmd := m.Update(key)
		_, ok := runCmd(t, cmd).(tea.QuitMsg)
		assert.True(t, ok, "quit keys must interrupt a drain")
	}
}

// The quit waits for the terminal's answer, so a fetch in flight can land
// after it. That page is dropped and no further fetch starts.
func TestPagerModelDropsBatchAfterQuit(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.hasPrinted = true
	m.fetching = true
	m.drainAll = true
	_, cmd := m.Update(teainput.Press("q"))
	_, ok := runCmd(t, cmd).(tea.QuitMsg)
	require.True(t, ok)

	_, cmd = m.Update(batchMsg{lines: []string{"6", "7"}, count: 2})
	assert.Nil(t, cmd, "a page that lands after quit is neither printed nor followed by a fetch")
	_, cmd = m.Update(batchMsg{err: errors.New("boom")})
	assert.Nil(t, cmd)
	assert.NoError(t, m.err, "an error that lands after quit is dropped too")
}

func TestPagerModelIgnoresKeysAfterDone(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 0}, 5)
	m.iterDone = true
	_, cmd := m.Update(teainput.Press("ctrl+c"))
	assert.Nil(t, cmd, "keys after completion should be no-ops")
}

func TestPagerModelViewHiddenUntilFirstBatch(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 10}, 5)
	assert.Empty(t, m.render(), "prompt must not flash before any output is rendered")
}

func TestPagerModelViewShowsSpinnerWhileFetching(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	assert.Contains(t, m.render(), pagerLoadingText)
	assert.NotContains(t, m.render(), pagerPromptText)
}

func TestPagerModelWindowSizeResizesPage(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 50)
	_, cmd := m.Update(tea.WindowSizeMsg{Height: 30, Width: 120})
	assert.Nil(t, cmd, "resize should not itself dispatch a command")
	assert.Equal(t, 30-pagerViewOverhead, m.pageSize)
}

func TestPagerModelWindowSizeFloorsAtMin(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 50)
	_, _ = m.Update(tea.WindowSizeMsg{Height: 3, Width: 80})
	assert.Equal(t, pagerMinPageSize, m.pageSize)
}
