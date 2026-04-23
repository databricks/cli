package cmdio

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/databricks-sdk-go/listing"
)

// pagerPageSize is the number of items rendered between prompts.
const pagerPageSize = 50

// pagerPromptText is shown between pages.
const pagerPromptText = "[space] more  [enter] all  [q|esc] quit"

// pagerModel drives the paged render loop through bubbletea. It owns the
// iterator, fetches one batch at a time via a tea.Cmd, emits rendered
// lines above the view with tea.Println, and shows the prompt while it
// waits for a key. Using bubbletea lets us skip manual raw-mode setup
// (tea enters/restores raw mode on its own), so we don't need a parallel
// CRLF translator for output written during raw mode.
type pagerModel[T any] struct {
	ctx      context.Context
	iter     listing.Iterator[T]
	pager    *templatePager
	pageSize int
	limit    int
	total    int

	// fetching tracks whether a fetchCmd is in flight. We only ever keep
	// one in flight at a time so the iterator isn't read from two
	// goroutines; if the user hits SPACE or ENTER while one is running,
	// we record the intent in drainAll and let the in-flight fetch
	// continue in drain mode when its batchMsg lands.
	fetching   bool
	drainAll   bool
	firstBatch bool
	iterDone   bool
	err        error
}

// batchMsg is delivered to Update when a fetched batch has been rendered
// into printable lines. done signals the iterator is exhausted (or the
// limit is reached); err surfaces iteration errors.
type batchMsg struct {
	lines []string
	done  bool
	err   error
}

func (m *pagerModel[T]) Init() tea.Cmd {
	m.fetching = true
	return m.fetchCmd()
}

// fetchCmd returns a tea.Cmd that reads one page from the iterator and
// renders it into lines. The command runs off the update loop so slow
// network fetches don't stall key handling.
func (m *pagerModel[T]) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		buf := make([]any, 0, m.pageSize)
		done := false
		for len(buf) < m.pageSize {
			if m.limit > 0 && m.total+len(buf) >= m.limit {
				done = true
				break
			}
			if !m.iter.HasNext(m.ctx) {
				done = true
				break
			}
			n, err := m.iter.Next(m.ctx)
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
}

func (m *pagerModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case batchMsg:
		m.fetching = false
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.firstBatch = true
		// Collapse the batch into a single Println so ordering is trivial:
		// tea.Println splits on \n internally, so one Cmd emits all rows.
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
			return m, tea.Sequence(printCmd, m.fetchCmd())
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

// handleKey routes a keystroke to the right state transition. Keys we
// don't care about are ignored so the user can mash the keyboard without
// affecting the pager.
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

// startAdvance handles SPACE: fetch one more page unless we're already
// fetching or draining. If a fetch is in flight we drop the keystroke.
func (m *pagerModel[T]) startAdvance() tea.Cmd {
	if m.drainAll || m.fetching {
		return nil
	}
	m.fetching = true
	return m.fetchCmd()
}

// startDrain handles ENTER: flip into drain-all mode. If a fetch is
// already in flight the batchMsg handler will see drainAll and continue
// fetching; otherwise we kick off the next fetch here.
func (m *pagerModel[T]) startDrain() tea.Cmd {
	if m.drainAll {
		return nil
	}
	m.drainAll = true
	if m.fetching {
		return nil
	}
	m.fetching = true
	return m.fetchCmd()
}

func (m *pagerModel[T]) View() string {
	if m.iterDone || m.drainAll || m.err != nil || !m.firstBatch {
		return ""
	}
	return pagerPromptText
}
