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

func TestPaginatedFetch(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*paginatedModel)
		msg    rowsFetchedMsg
		verify func(*testing.T, paginatedModel)
	}{
		{
			name:  "first batch",
			setup: func(*paginatedModel) {},
			msg:   rowsFetchedMsg{rows: [][]string{{"alice", "30"}, {"bob", "25"}}, exhausted: true},
			verify: func(t *testing.T, pm paginatedModel) {
				assert.Len(t, pm.rows, 2)
				assert.True(t, pm.exhausted)
				assert.Equal(t, 0, pm.cursor)
				assert.NotNil(t, pm.widths)
			},
		},
		{
			name: "subsequent batch appends",
			setup: func(m *paginatedModel) {
				m.rows = [][]string{{"alice", "30"}}
				m.widths = []int{5, 3}
			},
			msg: rowsFetchedMsg{rows: [][]string{{"bob", "25"}}, exhausted: false},
			verify: func(t *testing.T, pm paginatedModel) {
				assert.Len(t, pm.rows, 2)
				assert.False(t, pm.exhausted)
			},
		},
		{
			name:  "exhaustion with no rows",
			setup: func(*paginatedModel) {},
			msg:   rowsFetchedMsg{rows: nil, exhausted: true},
			verify: func(t *testing.T, pm paginatedModel) {
				assert.True(t, pm.exhausted)
				assert.Empty(t, pm.rows)
			},
		},
		{
			name:  "error",
			setup: func(*paginatedModel) {},
			msg:   rowsFetchedMsg{err: errors.New("network error")},
			verify: func(t *testing.T, pm paginatedModel) {
				require.Error(t, pm.err)
				assert.Equal(t, "network error", pm.err.Error())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, nil, 0)
			m.ready = true
			m.viewport.Width = 80
			m.viewport.Height = 20
			tt.setup(&m)
			result, _ := m.Update(tt.msg)
			tt.verify(t, result.(paginatedModel))
		})
	}
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

func TestPaginatedView(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*paginatedModel)
		want  string
	}{
		{"not ready", func(*paginatedModel) {}, "Loading..."},
		{"loading", func(m *paginatedModel) { m.ready = true; m.loading = true }, "Fetching results..."},
		{"no results", func(m *paginatedModel) { m.ready = true; m.exhausted = true }, "No results found."},
		{"error", func(m *paginatedModel) { m.ready = true; m.err = errors.New("something broke") }, "Error: something broke"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, nil, 0)
			tt.setup(&m)
			assert.Contains(t, m.View(), tt.want)
		})
	}
}

func TestMaybeFetch(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*paginatedModel)
		wantFetch bool
	}{
		{
			name: "triggers when near bottom and not loading",
			setup: func(m *paginatedModel) {
				m.rows = make([][]string, 15)
				m.cursor = 10
				m.loading = false
				m.exhausted = false
			},
			wantFetch: true,
		},
		{
			name: "does not trigger when exhausted",
			setup: func(m *paginatedModel) {
				m.rows = make([][]string, 15)
				m.cursor = 10
				m.exhausted = true
			},
			wantFetch: false,
		},
		{
			name: "does not trigger when loading",
			setup: func(m *paginatedModel) {
				m.rows = make([][]string, 15)
				m.cursor = 10
				m.loading = true
			},
			wantFetch: false,
		},
		{
			name: "does not trigger when searching",
			setup: func(m *paginatedModel) {
				m.rows = make([][]string, 15)
				m.cursor = 10
				m.search.active = true
			},
			wantFetch: false,
		},
		{
			name: "does not trigger when far from bottom",
			setup: func(m *paginatedModel) {
				m.rows = make([][]string, 50)
				m.cursor = 0
			},
			wantFetch: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, nil, 0)
			tt.setup(&m)
			m, cmd := maybeFetch(m)
			if tt.wantFetch {
				assert.NotNil(t, cmd)
				assert.True(t, m.loading, "loading should be true after fetch triggered")
			} else {
				assert.Nil(t, cmd)
			}
		})
	}
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
	m.search.active = true
	m.search.input = "test"

	// Submit search
	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm := result.(paginatedModel)

	assert.False(t, pm.search.active)
	assert.True(t, searchCalled)
	assert.NotNil(t, cmd)
	assert.NotNil(t, pm.search.saved)
	assert.Equal(t, 1, pm.fetchGeneration)

	// Restore by submitting empty search
	pm.search.active = true
	pm.search.input = ""
	pm.rows = [][]string{{"found:test"}}
	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm = result.(paginatedModel)

	assert.Equal(t, [][]string{{"original"}}, pm.rows)
	assert.Nil(t, pm.search.saved)
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

	m.search.active = true
	m.search.input = "test"

	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm := result.(paginatedModel)

	assert.NotNil(t, cmd)
	assert.NotNil(t, pm.search.saved)
	assert.Nil(t, pm.search.saved.rows)
	assert.Equal(t, 1, pm.fetchGeneration)

	pm.search.active = true
	pm.search.input = ""
	pm.rows = [][]string{{"found:test"}}
	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm = result.(paginatedModel)

	assert.Nil(t, pm.rows)
	assert.Equal(t, originalIter, pm.rowIter)
	assert.True(t, pm.exhausted)
	assert.Nil(t, pm.search.saved)
	assert.Equal(t, 2, pm.fetchGeneration)
}

func TestPaginatedSearchEscCancels(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.search.active = true
	m.search.input = "partial"
	m.viewport.Height = 20

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	assert.False(t, pm.search.active)
	assert.Equal(t, "", pm.search.input)
	assert.Equal(t, 21, pm.viewport.Height)
}

func TestPaginatedSearchInputKeys(t *testing.T) {
	tests := []struct {
		name      string
		initial   string
		key       tea.KeyMsg
		wantInput string
	}{
		{"backspace", "abc", tea.KeyMsg{Type: tea.KeyBackspace}, "ab"},
		{"typing", "", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, "a"},
		{"space", "my", tea.KeyMsg{Type: tea.KeySpace}, "my "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, nil, 0)
			m.search.active = true
			m.search.input = tt.initial
			result, cmd := m.updateSearch(tt.key)
			pm := result.(paginatedModel)
			assert.Equal(t, tt.wantInput, pm.search.input)
			assert.NotNil(t, cmd, "input change should schedule a debounce tick")
		})
	}
}

func TestPaginatedRenderFooter(t *testing.T) {
	searchCfg := &TableConfig{
		Columns: []ColumnDef{{Header: "Name"}},
		Search:  &SearchConfig{Placeholder: "type here"},
	}
	tests := []struct {
		name     string
		maxItems int
		cfg      *TableConfig
		setup    func(*paginatedModel)
		contains []string
	}{
		{
			name:     "exhausted",
			setup:    func(m *paginatedModel) { m.rows = [][]string{{"a", "1"}, {"b", "2"}}; m.exhausted = true },
			contains: []string{"2 rows", "q quit"},
		},
		{
			name:     "more available",
			setup:    func(m *paginatedModel) { m.rows = [][]string{{"a", "1"}}; m.exhausted = false },
			contains: []string{"more available"},
		},
		{
			name:     "limit reached",
			maxItems: 10,
			setup:    func(m *paginatedModel) { m.rows = make([][]string, 10); m.limitReached = true; m.exhausted = true },
			contains: []string{"limit: 10"},
		},
		{
			name:     "with search configured",
			cfg:      searchCfg,
			setup:    func(m *paginatedModel) { m.rows = [][]string{{"a"}} },
			contains: []string{"/ search"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, nil, tt.maxItems)
			if tt.cfg != nil {
				m.cfg = tt.cfg
			} else {
				m.cfg = newTestConfig()
			}
			tt.setup(&m)
			footer := m.renderFooter()
			for _, want := range tt.contains {
				assert.Contains(t, footer, want)
			}
		})
	}
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

func TestPaginatedSearchDebounceIncrementsSeq(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.search.active = true
	m.search.input = ""

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	pm := result.(paginatedModel)
	assert.Equal(t, 1, pm.search.debounceSeq)

	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	pm = result.(paginatedModel)
	assert.Equal(t, 2, pm.search.debounceSeq)
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
		search: searchState{
			active:      true,
			input:       "test",
			debounceSeq: 5,
		},
		ready: true,
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
		search: searchState{
			active:      true,
			input:       "hello",
			debounceSeq: 3,
		},
		rows:   [][]string{{"original"}},
		widths: []int{8},
		ready:  true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Send a matching debounce message (seq=3).
	result, cmd := m.Update(searchDebounceMsg{seq: 3})
	pm := result.(paginatedModel)

	assert.True(t, searchCalled)
	assert.NotNil(t, cmd, "should return fetch command")
	assert.NotNil(t, pm.search.saved)
	assert.Equal(t, [][]string{{"original"}}, pm.search.saved.rows)
}

func TestPaginatedSearchDebounceIgnoredWhenNotSearching(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.search.active = false
	m.search.debounceSeq = 1

	result, cmd := m.Update(searchDebounceMsg{seq: 1})
	pm := result.(paginatedModel)

	assert.Nil(t, cmd)
	assert.False(t, pm.search.active)
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
		search: searchState{
			active:      true,
			input:       "test",
			debounceSeq: 5,
		},
		ready: true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	result, cmd := m.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	pm := result.(paginatedModel)

	assert.True(t, searchCalled, "enter should trigger search immediately")
	assert.NotNil(t, cmd)
	assert.False(t, pm.search.active, "search mode should be exited")
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

	assert.True(t, pm.search.active)
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
		cfg:            cfg,
		headers:        []string{"Name"},
		rowIter:        &stringRowIterator{rows: [][]string{{"search-result"}}},
		makeFetchCmd:   newFetchCmdFunc(ctx),
		makeSearchIter: newSearchIterFunc(ctx, cfg.Search),
		search: searchState{
			active: true,
			input:  "test",
			saved: &savedSearch{
				rows:      [][]string{{"original"}},
				iter:      originalIter,
				exhausted: true,
			},
		},
		rows:            [][]string{{"search-result"}},
		widths:          []int{13},
		ready:           true,
		fetchGeneration: 2,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	assert.False(t, pm.search.active)
	assert.Equal(t, "", pm.search.input)
	assert.Equal(t, [][]string{{"original"}}, pm.rows)
	assert.Equal(t, originalIter, pm.rowIter)
	assert.True(t, pm.exhausted)
	assert.Nil(t, pm.search.saved)
	assert.Equal(t, 3, pm.fetchGeneration)
	assert.Equal(t, 0, pm.cursor)
}

func TestPaginatedSearchEscWithNoSearchStateDoesNothing(t *testing.T) {
	m := newTestModel(t, nil, 0)
	m.search.active = true
	m.search.input = "partial"
	m.rows = [][]string{{"data"}}
	m.viewport.Height = 20

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	assert.False(t, pm.search.active)
	assert.Equal(t, [][]string{{"data"}}, pm.rows, "rows should not change when there is no saved search state")
}

func TestPaginatedModelErr(t *testing.T) {
	m := newTestModel(t, nil, 0)
	assert.NoError(t, m.Err())

	m.err = errors.New("test error")
	assert.Equal(t, "test error", m.Err().Error())
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
	m.search.active = true
	m.search.input = ""
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
	assert.True(t, pm.search.active)
	assert.False(t, pm.loading, "loading should not be overloaded by search mode")

	// Cancel immediately with esc (no search executed).
	result, _ = pm.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm = result.(paginatedModel)

	assert.False(t, pm.search.active)
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
	m.search.active = true
	m.search.input = ""

	// User immediately cancels with esc (no search executed).
	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEscape})
	pm := result.(paginatedModel)

	// Generation must be unchanged so the in-flight fetch is accepted.
	assert.Equal(t, startGen, pm.fetchGeneration)
	assert.Nil(t, pm.search.saved)

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
		search: searchState{
			active:      true,
			input:       "",
			debounceSeq: 2,
			saved: &savedSearch{
				rows:      [][]string{{"original"}},
				iter:      originalIter,
				exhausted: true,
			},
		},
		rows:   [][]string{{"search-result"}},
		widths: []int{13},
		ready:  true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Debounce fires with empty search input, should restore.
	result, cmd := m.Update(searchDebounceMsg{seq: 2})
	pm := result.(paginatedModel)

	assert.Nil(t, cmd)
	assert.Equal(t, [][]string{{"original"}}, pm.rows)
	assert.Equal(t, originalIter, pm.rowIter)
	assert.Nil(t, pm.search.saved)
}
