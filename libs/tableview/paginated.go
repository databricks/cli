package tableview

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	fetchBatchSize           = 50
	fetchThresholdFromBottom = 10
	defaultMaxColumnWidth    = 50
	searchDebounceDelay      = 200 * time.Millisecond
)

// FinalModel is implemented by the paginated TUI model to expose errors
// that occurred during data fetching. tea.Program.Run() only returns
// framework errors, not application-level errors stored in the model.
type FinalModel interface {
	Err() error
}

// rowsFetchedMsg carries newly fetched rows from the iterator.
type rowsFetchedMsg struct {
	rows       [][]string
	exhausted  bool
	err        error
	generation int
}

// searchDebounceMsg fires after the debounce delay to trigger a search.
// The seq field is compared against the model's debounceSeq to discard stale ticks.
type searchDebounceMsg struct {
	seq int
}

type paginatedModel struct {
	cfg     *TableConfig
	headers []string

	viewport viewport.Model
	ready    bool

	// Data
	rows      [][]string
	loading   bool
	exhausted bool
	err       error

	// Fetch state
	rowIter         RowIterator
	makeFetchCmd    func(m paginatedModel) tea.Cmd // closure capturing ctx
	makeSearchIter  func(query string) RowIterator // closure capturing ctx
	fetchGeneration int

	// Display
	cursor int
	widths []int

	// Search
	search searchState

	// Limits
	maxItems     int
	limitReached bool
}

// searchState groups the server-side search / debounce state.
// When a search replaces the original iterator, saved holds the
// pre-search snapshot so it can be restored on cancel/clear.
type searchState struct {
	active      bool
	loading     bool
	input       string
	debounceSeq int
	saved       *savedSearch
}

type savedSearch struct {
	rows      [][]string
	iter      RowIterator
	exhausted bool
}

// Err returns the error recorded during data fetching, if any.
func (m paginatedModel) Err() error {
	return m.err
}

// newFetchCmdFunc returns a closure that creates fetch commands, capturing ctx.
func newFetchCmdFunc(ctx context.Context) func(paginatedModel) tea.Cmd {
	return func(m paginatedModel) tea.Cmd {
		iter := m.rowIter
		currentLen := len(m.rows)
		maxItems := m.maxItems
		generation := m.fetchGeneration

		return func() tea.Msg {
			var rows [][]string
			exhausted := false

			limit := fetchBatchSize
			if maxItems > 0 {
				remaining := maxItems - currentLen
				if remaining <= 0 {
					return rowsFetchedMsg{exhausted: true, generation: generation}
				}
				limit = min(limit, remaining)
			}

			for range limit {
				if !iter.HasNext(ctx) {
					if ctx.Err() != nil {
						return rowsFetchedMsg{err: ctx.Err(), generation: generation}
					}
					exhausted = true
					break
				}
				row, err := iter.Next(ctx)
				if err != nil {
					return rowsFetchedMsg{err: err, generation: generation}
				}
				rows = append(rows, row)
			}

			if maxItems > 0 && currentLen+len(rows) >= maxItems {
				exhausted = true
			}

			return rowsFetchedMsg{rows: rows, exhausted: exhausted, generation: generation}
		}
	}
}

// newSearchIterFunc returns a closure that creates search iterators, capturing ctx.
func newSearchIterFunc(ctx context.Context, search *SearchConfig) func(string) RowIterator {
	return func(query string) RowIterator {
		return search.NewIterator(ctx, query)
	}
}

// NewPaginatedProgram creates but does not run the paginated TUI program.
func NewPaginatedProgram(ctx context.Context, w io.Writer, cfg *TableConfig, iter RowIterator, maxItems int) *tea.Program {
	headers := make([]string, len(cfg.Columns))
	for i, col := range cfg.Columns {
		headers[i] = col.Header
	}

	m := paginatedModel{
		cfg:          cfg,
		headers:      headers,
		rowIter:      iter,
		makeFetchCmd: newFetchCmdFunc(ctx),
		maxItems:     maxItems,
		loading:      true, // Init() fires the first fetch
	}

	if cfg.Search != nil {
		m.makeSearchIter = newSearchIterFunc(ctx, cfg.Search)
	}

	return tea.NewProgram(m, tea.WithOutput(w))
}

func (m paginatedModel) Init() tea.Cmd {
	return m.makeFetchCmd(m)
}

func (m paginatedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fh := footerHeight
		if m.search.active {
			fh = searchFooterHeight
		}
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-fh)
			m.viewport.SetHorizontalStep(horizontalScrollStep)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - fh
		}
		if len(m.rows) > 0 {
			m.viewport.SetContent(m.renderContent())
		}
		return m, nil

	case rowsFetchedMsg:
		if msg.generation != m.fetchGeneration {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			// fetchGeneration is intentionally not bumped: scrolling past the
			// threshold retries the failed fetch, and the error is shown in
			// the footer until the retry succeeds.
			return m, nil
		}
		m.err = nil

		isFirstBatch := len(m.rows) == 0
		if m.search.loading {
			m.rows = msg.rows
			m.search.loading = false
			isFirstBatch = true
		} else {
			m.rows = append(m.rows, msg.rows...)
		}
		m.exhausted = msg.exhausted

		if m.maxItems > 0 && len(m.rows) >= m.maxItems {
			m.limitReached = true
			m.exhausted = true
		}

		if len(m.rows) > 0 {
			m.computeWidths()
			if isFirstBatch {
				m.cursor = 0
			}
		}

		if m.ready {
			m.viewport.SetContent(m.renderContent())
		}
		return m, nil

	case searchDebounceMsg:
		if msg.seq != m.search.debounceSeq || !m.search.active {
			return m, nil
		}
		return m.executeSearch(m.search.input)

	case tea.KeyMsg:
		if m.search.active {
			return m.updateSearch(msg)
		}
		return m.updateNormal(msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *paginatedModel) computeWidths() {
	caps := make([]int, len(m.headers))
	for i := range caps {
		maxW := defaultMaxColumnWidth
		if i < len(m.cfg.Columns) && m.cfg.Columns[i].MaxWidth > 0 {
			maxW = m.cfg.Columns[i].MaxWidth
		}
		caps[i] = maxW
	}
	m.widths = computeColumnWidths(m.headers, m.rows, caps)
}

func (m paginatedModel) renderContent() string {
	// Pre-truncate rows for display. MaxWidth truncation is destructive;
	// horizontal scroll won't recover hidden text.
	truncated := make([][]string, len(m.rows))
	for ri, row := range m.rows {
		vals := make([]string, len(m.headers))
		for i := range m.headers {
			if i < len(row) {
				v := row[i]
				maxW := defaultMaxColumnWidth
				if i < len(m.cfg.Columns) && m.cfg.Columns[i].MaxWidth > 0 {
					maxW = m.cfg.Columns[i].MaxWidth
				}
				if utf8.RuneCountInString(v) > maxW {
					runes := []rune(v)
					if maxW <= 3 {
						v = string(runes[:maxW])
					} else {
						v = string(runes[:maxW-3]) + "..."
					}
				}
				vals[i] = v
			}
		}
		truncated[ri] = vals
	}

	lines := renderTableToLines(m.headers, m.widths, truncated)

	// Apply cursor highlighting.
	result := make([]string, len(lines))
	for i, line := range lines {
		if i == m.cursor+headerLines {
			result[i] = cursorStyle.Render(line)
		} else {
			result[i] = line
		}
	}

	return strings.Join(result, "\n")
}

func (m paginatedModel) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "/":
		if m.cfg.Search != nil {
			m.search.active = true
			m.search.input = ""
			// Shrink viewport by one row to make room for the search input bar.
			m.viewport.Height--
			return m, nil
		}
		return m, nil
	case "up", "k":
		m.moveCursor(-1)
		m, cmd := maybeFetch(m)
		return m, cmd
	case "down", "j":
		m.moveCursor(1)
		m, cmd := maybeFetch(m)
		return m, cmd
	case "pgup", "b":
		m.moveCursor(-m.viewport.Height)
		return m, nil
	case "pgdown", "f", " ":
		m.moveCursor(m.viewport.Height)
		m, cmd := maybeFetch(m)
		return m, cmd
	case "g":
		m.cursor = 0
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoTop()
		return m, nil
	case "G":
		m.cursor = max(len(m.rows)-1, 0)
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoBottom()
		m, cmd := maybeFetch(m)
		return m, cmd
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *paginatedModel) moveCursor(delta int) {
	m.cursor += delta
	m.cursor = max(m.cursor, 0)
	m.cursor = min(m.cursor, max(len(m.rows)-1, 0))
	m.viewport.SetContent(m.renderContent())

	displayLine := m.cursor + headerLines
	scrollViewportToCursor(&m.viewport, displayLine)
}

func maybeFetch(m paginatedModel) (paginatedModel, tea.Cmd) {
	if m.loading || m.exhausted || m.search.active {
		return m, nil
	}
	if len(m.rows)-m.cursor <= fetchThresholdFromBottom {
		m.loading = true
		return m, m.makeFetchCmd(m)
	}
	return m, nil
}

// scheduleSearchDebounce returns a command that sends a searchDebounceMsg after the delay.
func (m *paginatedModel) scheduleSearchDebounce() tea.Cmd {
	m.search.debounceSeq++
	seq := m.search.debounceSeq
	return tea.Tick(searchDebounceDelay, func(_ time.Time) tea.Msg {
		return searchDebounceMsg{seq: seq}
	})
}

// restorePreSearchState restores the original (pre-search) data and resets
// loading so that maybeFetch is unblocked. Safe to call even when there is
// no saved search state.
func (m *paginatedModel) restorePreSearchState() {
	if m.search.saved != nil {
		// Bump generation to discard any in-flight search fetch, since we're
		// switching back to the original iterator.
		m.fetchGeneration++
		m.rows = m.search.saved.rows
		m.rowIter = m.search.saved.iter
		m.exhausted = m.search.saved.exhausted
		m.search.saved = nil
		m.limitReached = false
		m.loading = false
		m.search.loading = false
	}
	m.cursor = 0
	if m.ready {
		m.computeWidths()
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoTop()
	}
}

// executeSearch triggers a server-side search for the given query.
// If query is empty, it restores the original (pre-search) state.
func (m paginatedModel) executeSearch(query string) (tea.Model, tea.Cmd) {
	if query == "" {
		m.restorePreSearchState()
		return m, nil
	}

	if m.search.saved == nil {
		m.search.saved = &savedSearch{
			rows:      m.rows,
			iter:      m.rowIter,
			exhausted: m.exhausted,
		}
	}

	m.fetchGeneration++
	m.exhausted = false
	m.limitReached = false
	m.loading = true
	m.search.loading = true
	m.cursor = 0
	m.rowIter = m.makeSearchIter(query)
	return m, m.makeFetchCmd(m)
}

func (m paginatedModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.search.active = false
		// Restore viewport height now that search bar is hidden.
		m.viewport.Height++
		// Execute final search immediately (bypass debounce).
		return m.executeSearch(m.search.input)
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.search.active = false
		m.search.input = ""
		// Restore viewport height now that search bar is hidden.
		m.viewport.Height++
		m.restorePreSearchState()
		return m, nil
	case "backspace":
		if len(m.search.input) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.search.input)
			m.search.input = m.search.input[:len(m.search.input)-size]
		}
		return m, m.scheduleSearchDebounce()
	default:
		if msg.Type == tea.KeyRunes {
			m.search.input += msg.String()
			return m, m.scheduleSearchDebounce()
		}
		if msg.Type == tea.KeySpace {
			m.search.input += " "
			return m, m.scheduleSearchDebounce()
		}
		return m, nil
	}
}

func (m paginatedModel) View() string {
	if !m.ready {
		return "Loading..."
	}
	if len(m.rows) == 0 && m.loading {
		if m.search.loading {
			return "Searching..."
		}
		return "Fetching results..."
	}
	if len(m.rows) == 0 && m.exhausted {
		return "No results found."
	}
	if m.err != nil && len(m.rows) == 0 {
		return fmt.Sprintf("Error: %v", m.err)
	}

	footer := m.renderFooter()
	if m.err != nil {
		footer = footerStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}
	return m.viewport.View() + "\n" + footer
}

func (m paginatedModel) renderFooter() string {
	if m.search.active {
		placeholder := ""
		if m.cfg.Search != nil {
			placeholder = m.cfg.Search.Placeholder
		}
		input := m.search.input
		if input == "" && placeholder != "" {
			input = footerStyle.Render(placeholder)
		}
		prompt := searchStyle.Render("/ " + input + "█")
		status := fmt.Sprintf("%d rows loaded", len(m.rows))
		if m.search.loading {
			status = "Searching..."
		}
		return footerStyle.Render(status) + "\n" + prompt
	}

	var parts []string

	if m.limitReached {
		parts = append(parts, fmt.Sprintf("%d rows (limit: %d)", len(m.rows), m.maxItems))
	} else if m.exhausted {
		parts = append(parts, fmt.Sprintf("%d rows", len(m.rows)))
	} else {
		parts = append(parts, fmt.Sprintf("%d rows loaded (more available)", len(m.rows)))
	}

	if m.loading {
		parts = append(parts, "loading...")
	}

	parts = append(parts, "←→↑↓ scroll", "g/G top/bottom")

	if m.cfg.Search != nil {
		parts = append(parts, "/ search")
	}

	parts = append(parts, "q quit")

	if m.exhausted && len(m.rows) > 0 {
		pct := int(m.viewport.ScrollPercent() * 100)
		parts = append(parts, fmt.Sprintf("%d%%", pct))
	}

	return footerStyle.Render(strings.Join(parts, " | "))
}
