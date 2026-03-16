package tableview

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
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
	searching      bool
	searchInput    string
	debounceSeq    int
	hasSearchState bool
	savedRows      [][]string
	savedIter      RowIterator
	savedExhaust   bool

	// Limits
	maxItems     int
	limitReached bool
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
	}

	if cfg.Search != nil {
		m.makeSearchIter = newSearchIterFunc(ctx, cfg.Search)
	}

	return tea.NewProgram(m, tea.WithOutput(w))
}

// RunPaginated launches the paginated TUI table.
func RunPaginated(ctx context.Context, w io.Writer, cfg *TableConfig, iter RowIterator, maxItems int) error {
	p := NewPaginatedProgram(ctx, w, cfg, iter, maxItems)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if pm, ok := finalModel.(FinalModel); ok {
		if modelErr := pm.Err(); modelErr != nil {
			return modelErr
		}
	}
	return nil
}

// Err returns any error that occurred during data fetching.
func (m paginatedModel) Err() error {
	return m.err
}

func (m paginatedModel) Init() tea.Cmd {
	return m.makeFetchCmd(m)
}

func (m paginatedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fh := footerHeight
		if m.searching {
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
			return m, nil
		}

		isFirstBatch := len(m.rows) == 0
		m.rows = append(m.rows, msg.rows...)
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
		if msg.seq != m.debounceSeq || !m.searching {
			return m, nil
		}
		return m.executeSearch(m.searchInput)

	case tea.KeyMsg:
		if m.searching {
			return m.updateSearch(msg)
		}
		return m.updateNormal(msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *paginatedModel) computeWidths() {
	m.widths = make([]int, len(m.headers))
	for i, h := range m.headers {
		m.widths[i] = len(h)
	}
	for _, row := range m.rows {
		for i := range m.widths {
			if i < len(row) {
				maxW := defaultMaxColumnWidth
				if i < len(m.cfg.Columns) && m.cfg.Columns[i].MaxWidth > 0 {
					maxW = m.cfg.Columns[i].MaxWidth
				}
				m.widths[i] = min(max(m.widths[i], len(row[i])), maxW)
			}
		}
	}
}

func (m paginatedModel) renderContent() string {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, strings.Join(m.headers, "\t"))

	// Separator
	seps := make([]string, len(m.headers))
	for i, w := range m.widths {
		seps[i] = strings.Repeat("─", w)
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Data rows
	for _, row := range m.rows {
		vals := make([]string, len(m.headers))
		for i := range m.headers {
			if i < len(row) {
				v := row[i]
				maxW := defaultMaxColumnWidth
				if i < len(m.cfg.Columns) && m.cfg.Columns[i].MaxWidth > 0 {
					maxW = m.cfg.Columns[i].MaxWidth
				}
				if len(v) > maxW {
					if maxW <= 3 {
						v = v[:maxW]
					} else {
						v = v[:maxW-3] + "..."
					}
				}
				vals[i] = v
			}
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	tw.Flush()

	lines := strings.Split(buf.String(), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Apply cursor highlighting
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
			m.searching = true
			m.searchInput = ""
			// Prevent maybeFetch from starting new fetches against the old iterator
			// while we're in search mode. Any in-flight fetch will be discarded
			// via generation check when it returns.
			m.loading = true
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
	if m.loading || m.exhausted {
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
	m.debounceSeq++
	seq := m.debounceSeq
	return tea.Tick(searchDebounceDelay, func(_ time.Time) tea.Msg {
		return searchDebounceMsg{seq: seq}
	})
}

// restorePreSearchState restores the original (pre-search) data and resets
// loading so that maybeFetch is unblocked. Safe to call even when there is
// no saved search state.
func (m *paginatedModel) restorePreSearchState() {
	if m.hasSearchState {
		m.fetchGeneration++
		m.rows = m.savedRows
		m.rowIter = m.savedIter
		m.exhausted = m.savedExhaust
		m.hasSearchState = false
		m.savedRows = nil
		m.savedIter = nil
		m.savedExhaust = false
	}
	m.loading = false
	m.cursor = 0
	if m.ready {
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

	if !m.hasSearchState {
		m.hasSearchState = true
		m.savedRows = m.rows
		m.savedIter = m.rowIter
		m.savedExhaust = m.exhausted
	}

	m.fetchGeneration++
	m.rows = nil
	m.exhausted = false
	m.loading = true
	m.cursor = 0
	m.rowIter = m.makeSearchIter(query)
	return m, m.makeFetchCmd(m)
}

func (m paginatedModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searching = false
		m.viewport.Height++
		// Execute final search immediately (bypass debounce).
		return m.executeSearch(m.searchInput)
	case "esc", "ctrl+c":
		m.searching = false
		m.searchInput = ""
		m.viewport.Height++
		m.restorePreSearchState()
		return m, nil
	case "backspace":
		if len(m.searchInput) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.searchInput)
			m.searchInput = m.searchInput[:len(m.searchInput)-size]
		}
		return m, m.scheduleSearchDebounce()
	default:
		if msg.Type == tea.KeyRunes {
			m.searchInput += msg.String()
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
		return "Fetching results..."
	}
	if len(m.rows) == 0 && m.exhausted {
		return "No results found."
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	footer := m.renderFooter()
	return m.viewport.View() + "\n" + footer
}

func (m paginatedModel) renderFooter() string {
	if m.searching {
		placeholder := ""
		if m.cfg.Search != nil {
			placeholder = m.cfg.Search.Placeholder
		}
		input := m.searchInput
		if input == "" && placeholder != "" {
			input = footerStyle.Render(placeholder)
		}
		prompt := searchStyle.Render("/ " + input + "█")
		return footerStyle.Render(fmt.Sprintf("%d rows loaded", len(m.rows))) + "\n" + prompt
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
