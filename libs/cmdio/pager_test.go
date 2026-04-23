package cmdio

import (
	"errors"
	"reflect"
	"testing"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPager(t *testing.T, iter listing.Iterator[int], pageSize int) *pagerModel[int] {
	t.Helper()
	rowT, err := template.New("row").Funcs(renderFuncMap).Parse("{{range .}}{{.}}\n{{end}}")
	require.NoError(t, err)
	headerT, err := template.New("header").Funcs(renderFuncMap).Parse("")
	require.NoError(t, err)
	return &pagerModel[int]{
		ctx:  t.Context(),
		iter: iter,
		pager: &templatePager{
			headerT: headerT,
			rowT:    rowT,
		},
		pageSize: pageSize,
	}
}

// runCmd invokes a tea.Cmd and returns the message it produced. Fails the
// test if the cmd is nil.
func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	require.NotNil(t, cmd)
	return cmd()
}

// unwrapCmds returns the component cmds of a tea.Batch or tea.Sequence
// result. sequenceMsg is private to bubbletea; we identify it by the
// underlying []tea.Cmd slice kind and extract via reflect.
func unwrapCmds(t *testing.T, msg tea.Msg) []tea.Cmd {
	t.Helper()
	if bm, ok := msg.(tea.BatchMsg); ok {
		return []tea.Cmd(bm)
	}
	rv := reflect.ValueOf(msg)
	require.Equal(t, reflect.Slice, rv.Kind(), "expected a slice-of-cmds msg, got %T", msg)
	cmds := make([]tea.Cmd, rv.Len())
	for i := range cmds {
		c, ok := rv.Index(i).Interface().(tea.Cmd)
		require.True(t, ok, "slice element %d is not a tea.Cmd", i)
		cmds[i] = c
	}
	return cmds
}

// printedText returns the body of a tea.Println message. printLineMessage
// is private to bubbletea, so we pull the first string field via reflect.
func printedText(t *testing.T, msg tea.Msg) string {
	t.Helper()
	rv := reflect.ValueOf(msg)
	require.Equal(t, reflect.Struct, rv.Kind(), "expected a struct msg, got %T", msg)
	for i := range rv.NumField() {
		if rv.Field(i).Kind() == reflect.String {
			return rv.Field(i).String()
		}
	}
	t.Fatalf("no string field in %T", msg)
	return ""
}

func TestPagerModelInitFetchesFirstBatch(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 3}, 10)
	b, ok := runCmd(t, m.Init()).(batchMsg)
	require.True(t, ok, "init should produce a batchMsg")
	assert.True(t, b.done, "small iterator is drained in one batch")
	assert.Len(t, b.lines, 3)
}

func TestPagerModelBatchPrintsAndQuitsWhenDone(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 3}, 10)
	_, cmd := m.Update(batchMsg{lines: []string{"1", "2", "3"}, done: true})
	assert.True(t, m.iterDone)
	assert.True(t, m.firstBatch)
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
	_, cmd := m.Update(batchMsg{lines: []string{"1", "2"}, done: false})
	assert.False(t, m.iterDone)
	assert.True(t, m.firstBatch)
	assert.False(t, m.drainAll)
	assert.Equal(t, pagerPromptText, m.View())
	assert.Contains(t, printedText(t, runCmd(t, cmd)), "1\n2")
}

func TestPagerModelBatchDrainingChainsFetch(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.drainAll = true
	_, cmd := m.Update(batchMsg{lines: []string{"1", "2"}, done: false})
	cmds := unwrapCmds(t, runCmd(t, cmd))
	require.Len(t, cmds, 2)
	assert.Contains(t, printedText(t, runCmd(t, cmds[0])), "1\n2")
	_, isFetch := runCmd(t, cmds[1]).(batchMsg)
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
		key  tea.KeyMsg
	}{
		{"KeySpace", tea.KeyMsg{Type: tea.KeySpace}},
		{"KeyRunes space", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestPager(t, &numberIterator{n: 100}, 5)
			m.firstBatch = true
			_, cmd := m.Update(tc.key)
			_, ok := runCmd(t, cmd).(batchMsg)
			assert.True(t, ok, "space should fire a fetch")
		})
	}
}

func TestPagerModelEnterSetsDrainAll(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.firstBatch = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, m.drainAll)
	assert.Empty(t, m.View(), "no prompt while draining")
	_, ok := runCmd(t, cmd).(batchMsg)
	assert.True(t, ok, "enter should fire a fetch")
}

func TestPagerModelEnterIsNoOpWhileAlreadyDraining(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.firstBatch = true
	m.drainAll = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "re-entering drain shouldn't stack another fetch")
}

func TestPagerModelSpaceIgnoredDuringFetch(t *testing.T) {
	// Between Init and the first batchMsg, SPACE must not schedule a second
	// fetch: doing so would run the iterator from two goroutines at once.
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Nil(t, cmd, "SPACE while fetching must not dispatch another fetch")
}

func TestPagerModelEnterDuringFetchDefersFetch(t *testing.T) {
	// ENTER during an in-flight fetch flips drainAll but can't issue a new
	// fetch; the pending batchMsg will chain one when it arrives.
	m := newTestPager(t, &numberIterator{n: 100}, 5)
	m.fetching = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, m.drainAll)
	assert.Nil(t, cmd, "ENTER during fetch must defer to batchMsg chaining")
}

func TestPagerModelQuitKeys(t *testing.T) {
	cases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"Q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}}},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestPager(t, &numberIterator{n: 100}, 5)
			m.firstBatch = true
			_, cmd := m.Update(tc.key)
			_, ok := runCmd(t, cmd).(tea.QuitMsg)
			assert.True(t, ok)
		})
	}
}

func TestPagerModelQuitKeysInterruptDrain(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEsc},
		{Type: tea.KeyCtrlC},
	} {
		m := newTestPager(t, &numberIterator{n: 100}, 5)
		m.firstBatch = true
		m.drainAll = true
		_, cmd := m.Update(key)
		_, ok := runCmd(t, cmd).(tea.QuitMsg)
		assert.True(t, ok, "quit keys must interrupt a drain")
	}
}

func TestPagerModelIgnoresKeysAfterDone(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 0}, 5)
	m.iterDone = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.Nil(t, cmd, "keys after completion should be no-ops")
}

func TestPagerModelViewHiddenUntilFirstBatch(t *testing.T) {
	m := newTestPager(t, &numberIterator{n: 10}, 5)
	assert.Empty(t, m.View(), "prompt must not flash before any output is rendered")
}
