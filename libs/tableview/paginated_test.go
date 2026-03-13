package tableview

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringRowIterator struct {
	rows [][]string
	pos  int
}

func (s *stringRowIterator) HasNext(_ context.Context) bool {
	return s.pos < len(s.rows)
}

func (s *stringRowIterator) Next(_ context.Context) ([]string, error) {
	if s.pos >= len(s.rows) {
		return nil, errors.New("no more rows")
	}
	row := s.rows[s.pos]
	s.pos++
	return row, nil
}

func newTestConfig() *TableConfig {
	return &TableConfig{
		Columns: []ColumnDef{
			{Header: "Name"},
			{Header: "Age"},
		},
	}
}

func newTestModel(t *testing.T, rows [][]string, maxItems int) paginatedModel {
	iter := &stringRowIterator{rows: rows}
	cfg := newTestConfig()
	return paginatedModel{
		cfg:          cfg,
		headers:      []string{"Name", "Age"},
		rowIter:      iter,
		makeFetchCmd: newFetchCmdFunc(t.Context()),
		maxItems:     maxItems,
	}
}

func TestPaginatedModelInit(t *testing.T) {
	m := newTestModel(t, [][]string{{"alice", "30"}}, 0)
	cmd := m.Init()
	require.NotNil(t, cmd)
}

func TestPaginatedFetchFirstBatch(t *testing.T) {
	rows := [][]string{{"alice", "30"}, {"bob", "25"}}
	m := newTestModel(t, rows, 0)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20

	msg := rowsFetchedMsg{rows: rows, exhausted: true}
	result, _ := m.Update(msg)
	pm := result.(paginatedModel)

	assert.Len(t, pm.rows, 2)
	assert.True(t, pm.exhausted)
	assert.Equal(t, 0, pm.cursor)
	assert.NotNil(t, pm.widths)
}

func TestPaginatedFetchSubsequentBatch(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20
	m.rows = [][]string{{"alice", "30"}}
	m.widths = []int{5, 3}

	msg := rowsFetchedMsg{rows: [][]string{{"bob", "25"}}, exhausted: false}
	result, _ := m.Update(msg)
	pm := result.(paginatedModel)

	assert.Len(t, pm.rows, 2)
	assert.False(t, pm.exhausted)
}

func TestPaginatedFetchExhaustion(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20

	msg := rowsFetchedMsg{rows: nil, exhausted: true}
	result, _ := m.Update(msg)
	pm := result.(paginatedModel)

	assert.True(t, pm.exhausted)
	assert.Empty(t, pm.rows)
}

func TestPaginatedFetchError(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true

	msg := rowsFetchedMsg{err: errors.New("network error")}
	result, _ := m.Update(msg)
	pm := result.(paginatedModel)

	require.Error(t, pm.err)
	assert.Equal(t, "network error", pm.err.Error())
}

func TestPaginatedErrAccessor(t *testing.T) {
	m := newTestModel(t, nil, 0)
	assert.NoError(t, m.Err())

	m.err = errors.New("api timeout")
	assert.EqualError(t, m.Err(), "api timeout")
}

func TestPaginatedCursorMovement(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20
	m.rows = [][]string{{"alice", "30"}, {"bob", "25"}, {"charlie", "35"}}
	m.widths = []int{7, 3}
	m.cursor = 0

	// Move down
	m.moveCursor(1)
	assert.Equal(t, 1, m.cursor)

	// Move down again
	m.moveCursor(1)
	assert.Equal(t, 2, m.cursor)

	// Can't go past end
	m.moveCursor(1)
	assert.Equal(t, 2, m.cursor)

	// Move up
	m.moveCursor(-1)
	assert.Equal(t, 1, m.cursor)

	// Can't go above 0
	m.moveCursor(-5)
	assert.Equal(t, 0, m.cursor)
}

func TestPaginatedMaxItemsLimit(t *testing.T) {
	m := newTestModel(t, nil, 3)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20

	rows := [][]string{{"a", "1"}, {"b", "2"}, {"c", "3"}}
	msg := rowsFetchedMsg{rows: rows, exhausted: false}
	result, _ := m.Update(msg)
	pm := result.(paginatedModel)

	assert.True(t, pm.limitReached)
	assert.True(t, pm.exhausted)
	assert.Len(t, pm.rows, 3)
}

func TestPaginatedViewLoading(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.loading = true
	view := m.View()
	assert.Equal(t, "Fetching results...", view)
}

func TestPaginatedViewNoResults(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.exhausted = true
	view := m.View()
	assert.Equal(t, "No results found.", view)
}

func TestPaginatedViewError(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.err = errors.New("something broke")
	view := m.View()
	assert.Contains(t, view, "Error: something broke")
}

func TestPaginatedViewNotReady(t *testing.T) {
	m := newTestModel(t, nil, 0)
	view := m.View()
	assert.Equal(t, "Loading...", view)
}

func TestPaginatedMaybeFetchTriggered(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.rows = make([][]string, 15)
	m.cursor = 10
	m.loading = false
	m.exhausted = false

	m, cmd := maybeFetch(m)
	assert.NotNil(t, cmd)
	assert.True(t, m.loading, "loading should be true after fetch triggered")
}

func TestPaginatedMaybeFetchNotTriggeredWhenExhausted(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.rows = make([][]string, 15)
	m.cursor = 10
	m.exhausted = true

	_, cmd := maybeFetch(m)
	assert.Nil(t, cmd)
}

func TestPaginatedMaybeFetchNotTriggeredWhenLoading(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.rows = make([][]string, 15)
	m.cursor = 10
	m.loading = true

	_, cmd := maybeFetch(m)
	assert.Nil(t, cmd)
}

func TestPaginatedMaybeFetchNotTriggeredWhenFarFromBottom(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.rows = make([][]string, 50)
	m.cursor = 0

	_, cmd := maybeFetch(m)
	assert.Nil(t, cmd)
}

func TestPaginatedSearchEnterAndRestore(t *testing.T) {
	searchCalled := false
	cfg := &TableConfig{
		Columns: []ColumnDef{
			{Header: "Name"},
		},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, query string) RowIterator {
				searchCalled = true
				return &stringRowIterator{rows: [][]string{{"found:" + query}}}
			},
		},
	}

	ctx := t.Context()
	m := paginatedModel{
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        &stringRowIterator{rows: [][]string{{"original"}}},
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		rows:           [][]string{{"original"}},
		widths:         []int{8},
		ready:          true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Enter search mode
	m.searching = true
	m.searchInput = "test"

	// Submit search
	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm := result.(paginatedModel)

	assert.False(t, pm.searching)
	assert.True(t, searchCalled)
	assert.NotNil(t, cmd)
	assert.NotNil(t, pm.savedRows)

	// Restore by submitting empty search
	pm.searching = true
	pm.searchInput = ""
	pm.rows = [][]string{{"found:test"}}
	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm = result.(paginatedModel)

	assert.Equal(t, [][]string{{"original"}}, pm.rows)
	assert.Nil(t, pm.savedRows)
}

func TestPaginatedSearchEscCancels(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = "partial"
	m.viewport.Height = 20

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	assert.False(t, pm.searching)
	assert.Equal(t, "", pm.searchInput)
	assert.Equal(t, 21, pm.viewport.Height)
}

func TestPaginatedSearchBackspace(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = "abc"

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyBackspace})
	pm := result.(paginatedModel)

	assert.Equal(t, "ab", pm.searchInput)
}

func TestPaginatedSearchTyping(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = ""

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	pm := result.(paginatedModel)

	assert.Equal(t, "a", pm.searchInput)
}

func TestPaginatedRenderFooterExhausted(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.rows = [][]string{{"a", "1"}, {"b", "2"}}
	m.exhausted = true
	m.cfg = newTestConfig()
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20

	footer := m.renderFooter()
	assert.Contains(t, footer, "2 rows")
	assert.Contains(t, footer, "q quit")
}

func TestPaginatedRenderFooterMoreAvailable(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.rows = [][]string{{"a", "1"}}
	m.exhausted = false
	m.cfg = newTestConfig()

	footer := m.renderFooter()
	assert.Contains(t, footer, "more available")
}

func TestPaginatedRenderFooterLimitReached(t *testing.T) {
	m := newTestModel(t, nil, 10)
	m.rows = make([][]string, 10)
	m.limitReached = true
	m.exhausted = true
	m.cfg = newTestConfig()

	footer := m.renderFooter()
	assert.Contains(t, footer, "limit: 10")
}

func TestMaybeFetchSetsLoadingAndPreventsDoubleFetch(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Simulate a first batch loaded with more available.
	rows := make([][]string, 15)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("name%d", i), strconv.Itoa(i)}
	}
	msg := rowsFetchedMsg{rows: rows, exhausted: false}
	result, _ := m.Update(msg)
	m = result.(paginatedModel)

	// Move cursor near bottom to trigger fetch threshold.
	m.cursor = len(m.rows) - 5
	m.viewport.SetContent(m.renderContent())

	// Trigger update with down key.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	um := updated.(paginatedModel)

	require.NotNil(t, cmd, "fetch should be triggered when near bottom")
	assert.True(t, um.loading, "model should be in loading state when fetch triggered")

	// Second down key should NOT trigger another fetch while loading.
	updated2, cmd2 := um.Update(tea.KeyMsg{Type: tea.KeyDown})
	_ = updated2
	assert.Nil(t, cmd2, "should not trigger second fetch while loading")
}

func TestFetchCmdWithIterator(t *testing.T) {
	rows := make([][]string, 60)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("name%d", i), strconv.Itoa(i)}
	}
	m := newTestModel(t, rows, 0)

	// Init returns the first fetch command.
	cmd := m.Init()
	require.NotNil(t, cmd)

	// Execute the command to get the message.
	msg := cmd()
	fetched, ok := msg.(rowsFetchedMsg)
	require.True(t, ok)

	assert.NoError(t, fetched.err)
	assert.Len(t, fetched.rows, fetchBatchSize)
	assert.False(t, fetched.exhausted, "iterator should have more rows")
}

func TestFetchCmdExhaustsSmallIterator(t *testing.T) {
	rows := [][]string{{"alice", "30"}, {"bob", "25"}}
	m := newTestModel(t, rows, 0)

	cmd := m.Init()
	require.NotNil(t, cmd)

	msg := cmd()
	fetched, ok := msg.(rowsFetchedMsg)
	require.True(t, ok)

	assert.NoError(t, fetched.err)
	assert.Len(t, fetched.rows, 2)
	assert.True(t, fetched.exhausted, "small iterator should be exhausted")
}

func TestPaginatedRenderFooterWithSearch(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.cfg = &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search:  &SearchConfig{Placeholder: "type here"},
	}
	m.rows = [][]string{{"a"}}

	footer := m.renderFooter()
	assert.Contains(t, footer, "/ search")
}
