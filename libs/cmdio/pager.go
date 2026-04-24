package cmdio

import (
	"context"
	"strings"
	"time"

	bubblespinner "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/databricks-sdk-go/listing"
)

// pagerFallbackPageSize is used before the first WindowSizeMsg arrives,
// and when the terminal height is too small to size a page by itself.
const pagerFallbackPageSize = 50

// pagerMinPageSize is the floor: one line of header plus a few rows so
// the prompt still has something to sit under.
const pagerMinPageSize = 5

// pagerViewOverhead is the number of lines we keep below the printed
// rows for the prompt (or spinner).
const pagerViewOverhead = 1

// pagerPromptText is shown between pages.
const pagerPromptText = "[space] more  [enter] all  [q|esc] quit"

// pagerLoadingText is appended to the spinner while a fetch is in flight.
const pagerLoadingText = "loading..."

// pagerModel is the tea.Model that drives the paged render loop: one
// fetch produces a batchMsg, Update prints it via tea.Println, and
// View shows the prompt between pages.
type pagerModel[T any] struct {
	iter    listing.Iterator[T]
	pager   *templatePager
	spinner bubblespinner.Model
	// fetch is bound at construction with the caller's context captured
	// so we don't have to stash ctx on the struct (tea.Cmd has no ctx
	// parameter of its own).
	fetch    func() tea.Msg
	pageSize int
	limit    int
	total    int

	// Keep only one fetch in flight at a time: the iterator is not safe
	// to read from two goroutines. If SPACE or ENTER arrives while
	// fetching, drainAll is recorded and the pending batchMsg chains
	// the next fetch.
	fetching   bool
	drainAll   bool
	hasPrinted bool
	iterDone   bool
	err        error
}

// newPagerModel wires ctx into the fetch closure so nothing on the
// struct has to hold onto a context.
func newPagerModel[T any](
	ctx context.Context,
	iter listing.Iterator[T],
	pager *templatePager,
	pageSize, limit int,
) *pagerModel[T] {
	m := &pagerModel[T]{
		iter:     iter,
		pager:    pager,
		spinner:  newPagerSpinner(),
		pageSize: pageSize,
		limit:    limit,
	}
	m.fetch = func() tea.Msg { return m.doFetch(ctx) }
	return m
}

// newPagerSpinner builds a spinner matching the one the cmdio package's
// NewSpinner uses, so interactive feedback looks the same everywhere.
func newPagerSpinner() bubblespinner.Model {
	s := bubblespinner.New()
	s.Spinner = bubblespinner.Spinner{
		Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		FPS:    time.Second / 5,
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	return s
}

// batchMsg carries the rendered lines from one fetch. done is true when
// the iterator is exhausted or the limit is reached.
type batchMsg struct {
	lines []string
	done  bool
	err   error
}

func (m *pagerModel[T]) Init() tea.Cmd {
	m.fetching = true
	return tea.Batch(m.fetch, m.spinner.Tick)
}

// doFetch reads one page from the iterator and renders it into lines.
// It runs off the update loop so a slow network fetch doesn't stall
// key handling.
func (m *pagerModel[T]) doFetch(ctx context.Context) tea.Msg {
	buf := make([]any, 0, m.pageSize)
	done := false
	for len(buf) < m.pageSize {
		if m.limit > 0 && m.total+len(buf) >= m.limit {
			done = true
			break
		}
		if !m.iter.HasNext(ctx) {
			done = true
			break
		}
		n, err := m.iter.Next(ctx)
		if err != nil {
			return batchMsg{err: err}
		}
		buf = append(buf, n)
	}
	lines, err := m.pager.flushLines(buf)
	if err != nil {
		return batchMsg{err: err}
	}
	m.total += len(buf)
	return batchMsg{lines: lines, done: done}
}

func (m *pagerModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.pageSize = max(msg.Height-pagerViewOverhead, pagerMinPageSize)
		return m, nil

	case bubblespinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case batchMsg:
		m.fetching = false
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.hasPrinted = true
		// One Println cmd (not N) keeps the batch ordered even though
		// tea.Sequence dispatches each cmd on its own goroutine.
		var printCmd tea.Cmd
		if len(msg.lines) > 0 {
			printCmd = tea.Println(strings.Join(msg.lines, "\n"))
		}
		switch {
		case msg.done:
			m.iterDone = true
			return m, tea.Sequence(printCmd, tea.Quit)
		case m.drainAll:
			m.fetching = true
			return m, tea.Sequence(printCmd, m.fetch)
		default:
			return m, printCmd
		}

	case tea.KeyMsg:
		if m.iterDone {
			return m, nil
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *pagerModel[T]) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive // the pager only cares about a few keys
	case tea.KeyEnter:
		return m, m.startDrain()
	case tea.KeyEsc, tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeySpace:
		return m, m.startAdvance()
	case tea.KeyRunes:
		switch msg.String() {
		case " ":
			return m, m.startAdvance()
		case "q", "Q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *pagerModel[T]) startAdvance() tea.Cmd {
	if m.drainAll || m.fetching {
		return nil
	}
	m.fetching = true
	return m.fetch
}

func (m *pagerModel[T]) startDrain() tea.Cmd {
	if m.drainAll {
		return nil
	}
	m.drainAll = true
	// If a fetch is already in flight, its batchMsg will see drainAll
	// and chain the next fetch. Otherwise kick one off here.
	if m.fetching {
		return nil
	}
	m.fetching = true
	return m.fetch
}

func (m *pagerModel[T]) View() string {
	switch {
	case m.iterDone || m.err != nil:
		return ""
	case m.fetching:
		return m.spinner.View() + " " + pagerLoadingText
	case m.drainAll || !m.hasPrinted:
		return ""
	default:
		return pagerPromptText
	}
}
