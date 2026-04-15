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

// TestKeyBeforeFirstFetchDoesNotDoubleFetch verifies that a keypress arriving
// before the first fetch completes does not trigger a second concurrent fetch.
// This guards against a data race on the shared iterator.
func TestKeyBeforeFirstFetchDoesNotDoubleFetch(t *testing.T) {
	rows := [][]string{{"alice", "30"}, {"bob", "25"}}
	iter := &stringRowIterator{rows: rows}
	cfg := newTestConfig()

	// Construct the model the same way NewPaginatedProgram does,
	// with loading=true since Init() fires the first fetch.
	m := paginatedModel{
		cfg:          cfg,
		headers:      []string{"Name", "Age"},
		rowIter:      iter,
		makeFetchCmd: newFetchCmdFunc(t.Context()),
		loading:      true,
		ready:        true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Simulate a keypress arriving before the first rowsFetchedMsg.
	// With loading=true, maybeFetch must bail out.
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	pm := result.(paginatedModel)

	assert.Nil(t, cmd, "must not trigger a second fetch while the initial fetch is in-flight")
	assert.True(t, pm.loading, "loading must remain true")
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

func TestPaginatedMaybeFetchNotTriggeredWhenSearching(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.rows = make([][]string, 15)
	m.cursor = 10
	m.searching = true

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
	assert.True(t, pm.hasSearchState)
	assert.Equal(t, 1, pm.fetchGeneration)

	// Restore by submitting empty search
	pm.searching = true
	pm.searchInput = ""
	pm.rows = [][]string{{"found:test"}}
	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm = result.(paginatedModel)

	assert.Equal(t, [][]string{{"original"}}, pm.rows)
	assert.False(t, pm.hasSearchState)
	assert.Nil(t, pm.savedRows)
	assert.Equal(t, 2, pm.fetchGeneration)
}

func TestPaginatedSearchRestoreEmptyOriginalTable(t *testing.T) {
	cfg := &TableConfig{
		Columns: []ColumnDef{
			{Header: "Name"},
		},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, query string) RowIterator {
				return &stringRowIterator{rows: [][]string{{"found:" + query}}}
			},
		},
	}

	ctx := t.Context()
	originalIter := &stringRowIterator{}
	m := paginatedModel{
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        originalIter,
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		exhausted:      true,
		ready:          true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	m.searching = true
	m.searchInput = "test"

	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm := result.(paginatedModel)

	assert.NotNil(t, cmd)
	assert.True(t, pm.hasSearchState)
	assert.Nil(t, pm.savedRows)
	assert.Equal(t, 1, pm.fetchGeneration)

	pm.searching = true
	pm.searchInput = ""
	pm.rows = [][]string{{"found:test"}}
	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm = result.(paginatedModel)

	assert.Nil(t, pm.rows)
	assert.Equal(t, originalIter, pm.rowIter)
	assert.True(t, pm.exhausted)
	assert.False(t, pm.hasSearchState)
	assert.Equal(t, 2, pm.fetchGeneration)
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

	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyBackspace})
	pm := result.(paginatedModel)

	assert.Equal(t, "ab", pm.searchInput)
	assert.NotNil(t, cmd, "backspace should schedule a debounce tick")
}

func TestPaginatedSearchTyping(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = ""

	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	pm := result.(paginatedModel)

	assert.Equal(t, "a", pm.searchInput)
	assert.NotNil(t, cmd, "typing should schedule a debounce tick")
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

func TestPaginatedIgnoresStaleFetchMessages(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20
	m.rows = [][]string{{"search", "1"}}
	m.widths = []int{6, 1}
	m.loading = true
	m.fetchGeneration = 1

	result, _ := m.Update(rowsFetchedMsg{
		rows:       [][]string{{"stale", "2"}},
		exhausted:  true,
		generation: 0,
	})
	pm := result.(paginatedModel)

	assert.Equal(t, [][]string{{"search", "1"}}, pm.rows)
	assert.False(t, pm.exhausted)
	assert.True(t, pm.loading)
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
	assert.Equal(t, 0, fetched.generation)
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
	assert.Equal(t, 0, fetched.generation)
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

func TestPaginatedSearchDebounceIncrementsSeq(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = ""

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	pm := result.(paginatedModel)
	assert.Equal(t, 1, pm.debounceSeq)

	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	pm = result.(paginatedModel)
	assert.Equal(t, 2, pm.debounceSeq)
}

func TestPaginatedSearchDebounceStaleTickIgnored(t *testing.T) {
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, _ string) RowIterator {
				t.Error("search should not be called for stale debounce")
				return &stringRowIterator{}
			},
		},
	}

	ctx := t.Context()
	m := paginatedModel{
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        &stringRowIterator{},
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		searching:      true,
		searchInput:    "test",
		debounceSeq:    5,
		ready:          true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Send a stale debounce message (seq=3, current=5).
	result, cmd := m.Update(searchDebounceMsg{seq: 3})
	pm := result.(paginatedModel)

	assert.Nil(t, cmd)
	assert.Nil(t, pm.rows, "rows should not change for stale debounce")
}

func TestPaginatedSearchDebounceCurrentSeqTriggers(t *testing.T) {
	searchCalled := false
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, query string) RowIterator {
				searchCalled = true
				assert.Equal(t, "hello", query)
				return &stringRowIterator{rows: [][]string{{"found"}}}
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
		searching:      true,
		searchInput:    "hello",
		debounceSeq:    3,
		rows:           [][]string{{"original"}},
		widths:         []int{8},
		ready:          true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Send a matching debounce message (seq=3).
	result, cmd := m.Update(searchDebounceMsg{seq: 3})
	pm := result.(paginatedModel)

	assert.True(t, searchCalled)
	assert.NotNil(t, cmd, "should return fetch command")
	assert.True(t, pm.hasSearchState)
	assert.Equal(t, [][]string{{"original"}}, pm.savedRows)
}

func TestPaginatedSearchDebounceIgnoredWhenNotSearching(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = false
	m.debounceSeq = 1

	result, cmd := m.Update(searchDebounceMsg{seq: 1})
	pm := result.(paginatedModel)

	assert.Nil(t, cmd)
	assert.False(t, pm.searching)
}

func TestPaginatedSearchEnterBypassesDebounce(t *testing.T) {
	searchCalled := false
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
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
		searching:      true,
		searchInput:    "test",
		debounceSeq:    5,
		ready:          true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm := result.(paginatedModel)

	assert.True(t, searchCalled, "enter should trigger search immediately")
	assert.NotNil(t, cmd)
	assert.False(t, pm.searching, "search mode should be exited")
}

func TestPaginatedSearchModeBlocksFetch(t *testing.T) {
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, _ string) RowIterator {
				return &stringRowIterator{}
			},
		},
	}

	ctx := t.Context()
	m := paginatedModel{
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        &stringRowIterator{rows: make([][]string, 20)},
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		rows:           make([][]string, 15),
		widths:         []int{4},
		ready:          true,
		loading:        false,
		exhausted:      false,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Enter search mode via "/" key.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	pm := result.(paginatedModel)

	assert.True(t, pm.searching)
	assert.False(t, pm.loading, "entering search mode should not overload loading flag")

	// Verify maybeFetch is blocked by the searching flag.
	pm.cursor = len(pm.rows) - 1
	pm, cmd := maybeFetch(pm)
	assert.Nil(t, cmd, "maybeFetch should not trigger while searching is true")
}

func TestPaginatedSearchExecuteSetsLoading(t *testing.T) {
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, _ string) RowIterator {
				return &stringRowIterator{rows: [][]string{{"result"}}}
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

	result, cmd := m.executeSearch("test")
	pm := result.(paginatedModel)

	assert.NotNil(t, cmd)
	assert.True(t, pm.loading, "executeSearch should set loading=true to prevent overlapping fetches")
}

func TestPaginatedSearchEscRestoresData(t *testing.T) {
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, _ string) RowIterator {
				return &stringRowIterator{rows: [][]string{{"search-result"}}}
			},
		},
	}

	ctx := t.Context()
	originalIter := &stringRowIterator{rows: [][]string{{"original"}}}
	m := paginatedModel{
		cfg:             cfg,
		headers:         []string{"Name"},
		rowIter:         &stringRowIterator{rows: [][]string{{"search-result"}}},
		makeFetchCmd:    newFetchCmdFunc(ctx),
		makeSearchIter:  newSearchIterFunc(ctx, cfg.Search),
		searching:       true,
		searchInput:     "test",
		hasSearchState:  true,
		savedRows:       [][]string{{"original"}},
		savedIter:       originalIter,
		savedExhaust:    true,
		rows:            [][]string{{"search-result"}},
		widths:          []int{13},
		ready:           true,
		fetchGeneration: 2,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	assert.False(t, pm.searching)
	assert.Equal(t, "", pm.searchInput)
	assert.Equal(t, [][]string{{"original"}}, pm.rows)
	assert.Equal(t, originalIter, pm.rowIter)
	assert.True(t, pm.exhausted)
	assert.False(t, pm.hasSearchState)
	assert.Nil(t, pm.savedRows)
	assert.Equal(t, 3, pm.fetchGeneration)
	assert.Equal(t, 0, pm.cursor)
}

func TestPaginatedSearchEscWithNoSearchStateDoesNothing(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = "partial"
	m.rows = [][]string{{"data"}}
	m.viewport.Height = 20

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	assert.False(t, pm.searching)
	assert.Equal(t, [][]string{{"data"}}, pm.rows, "rows should not change when there is no saved search state")
}

func TestPaginatedModelErr(t *testing.T) {
	m := newTestModel(t, nil, 0)
	assert.NoError(t, m.Err())

	m.err = errors.New("test error")
	assert.Equal(t, "test error", m.Err().Error())
}

func TestPaginatedSearchSpaceCharacterInput(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = "my"

	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeySpace})
	pm := result.(paginatedModel)

	assert.Equal(t, "my ", pm.searchInput)
	assert.NotNil(t, cmd, "space should schedule a debounce tick")
}

func TestPaginatedFetchErrorClearedOnSuccess(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Simulate a fetch error.
	errMsg := rowsFetchedMsg{err: errors.New("transient network error")}
	result, _ := m.Update(errMsg)
	pm := result.(paginatedModel)
	require.Error(t, pm.err)

	// Simulate a successful fetch afterward.
	successMsg := rowsFetchedMsg{rows: [][]string{{"alice", "30"}}, exhausted: true}
	result, _ = pm.Update(successMsg)
	pm = result.(paginatedModel)

	assert.NoError(t, pm.err, "error should be cleared after successful fetch")
	assert.Len(t, pm.rows, 1)
}

func TestPaginatedSearchEscWithoutSearchStateKeepsGeneration(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.searching = true
	m.searchInput = ""
	m.loading = true // fetch was in-flight before entering search
	m.viewport.Height = 20
	m.fetchGeneration = 5

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	assert.Equal(t, 5, pm.fetchGeneration, "fetchGeneration should NOT be bumped without search state")
	assert.True(t, pm.loading, "loading should be preserved when hasSearchState is false and a fetch was in-flight")
}

func TestPaginatedSearchEscWithoutExecutingUnblocksFetch(t *testing.T) {
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, _ string) RowIterator {
				return &stringRowIterator{}
			},
		},
	}

	ctx := t.Context()
	m := paginatedModel{
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        &stringRowIterator{rows: make([][]string, 20)},
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		rows:           make([][]string, 15),
		widths:         []int{4},
		ready:          true,
		loading:        false,
		exhausted:      false,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Enter search mode via "/".
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	pm := result.(paginatedModel)
	assert.True(t, pm.searching)
	assert.False(t, pm.loading, "loading should not be overloaded by search mode")

	// Cancel immediately with esc (no search executed).
	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm = result.(paginatedModel)

	assert.False(t, pm.searching)
	assert.False(t, pm.loading, "loading should remain false after esc")

	// Verify maybeFetch can fire again (searching=false, loading=false).
	pm.cursor = len(pm.rows) - 1
	pm, cmd := maybeFetch(pm)
	assert.NotNil(t, cmd, "maybeFetch should trigger after search mode is exited")
}

func TestPaginatedSearchEscBeforeFetchCompletesKeepsRows(t *testing.T) {
	ctx := t.Context()
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, _ string) RowIterator {
				return &stringRowIterator{}
			},
		},
	}

	iter := &stringRowIterator{rows: [][]string{{"row1"}, {"row2"}}}
	m := paginatedModel{
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        iter,
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		widths:         []int{4},
		ready:          true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Simulate: a fetch is in-flight at generation 0.
	m.loading = true
	startGen := m.fetchGeneration

	// User enters search mode (pressing "/").
	m.searching = true
	m.searchInput = ""

	// User immediately cancels with esc (no search executed).
	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	// Generation must be unchanged so the in-flight fetch is accepted.
	assert.Equal(t, startGen, pm.fetchGeneration)
	assert.False(t, pm.hasSearchState)

	// Simulate the in-flight fetch completing with the original generation.
	fetched := rowsFetchedMsg{
		rows:       [][]string{{"fetched-row"}},
		exhausted:  true,
		generation: startGen,
	}
	result2, _ := pm.Update(fetched)
	pm2 := result2.(paginatedModel)

	// The rows must be accepted, not silently dropped.
	assert.Equal(t, [][]string{{"fetched-row"}}, pm2.rows)
	assert.True(t, pm2.exhausted)
}

func TestPaginatedSearchDebounceEmptyQueryRestores(t *testing.T) {
	cfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search: &SearchConfig{
			Placeholder: "search...",
			NewIterator: func(_ context.Context, _ string) RowIterator {
				return &stringRowIterator{}
			},
		},
	}

	ctx := t.Context()
	originalIter := &stringRowIterator{rows: [][]string{{"original"}}}
	m := paginatedModel{
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        &stringRowIterator{rows: [][]string{{"search-result"}}},
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		searching:      true,
		searchInput:    "",
		debounceSeq:    2,
		hasSearchState: true,
		savedRows:      [][]string{{"original"}},
		savedIter:      originalIter,
		savedExhaust:   true,
		rows:           [][]string{{"search-result"}},
		widths:         []int{13},
		ready:          true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Debounce fires with empty search input, should restore.
	result, cmd := m.Update(searchDebounceMsg{seq: 2})
	pm := result.(paginatedModel)

	assert.Nil(t, cmd)
	assert.Equal(t, [][]string{{"original"}}, pm.rows)
	assert.Equal(t, originalIter, pm.rowIter)
	assert.False(t, pm.hasSearchState)
}
