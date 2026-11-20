package cmdio

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplyGuardDefersQuitUntilCursorReport(t *testing.T) {
	var mu sync.Mutex
	var sent []tea.Msg
	g := &replyGuard{wait: time.Hour, send: func(m tea.Msg) {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, m)
	}}
	sentMsgs := func() []tea.Msg {
		mu.Lock()
		defer mu.Unlock()
		return append([]tea.Msg(nil), sent...)
	}

	assert.Nil(t, g.filter(nil, tea.QuitMsg{}), "the first quit is held")
	require.Eventually(t, func() bool { return len(sentMsgs()) == 1 }, time.Second, time.Millisecond)
	assert.Equal(t, tea.RequestCursorPosition(), sentMsgs()[0])

	assert.Nil(t, g.filter(nil, tea.QuitMsg{}), "a second quit while waiting is dropped")
	assert.Nil(t, g.filter(nil, tea.KeyPressMsg{Code: 'a', Text: "a"}), "keys typed after quitting are dropped")
	assert.NotNil(t, g.filter(nil, tea.WindowSizeMsg{}), "other messages still reach the program")

	assert.Nil(t, g.filter(nil, tea.CursorPositionMsg{}), "the report is consumed")
	assert.False(t, g.timeout.Stop(), "the report stops the timeout")
	require.Eventually(t, func() bool { return len(sentMsgs()) == 2 }, time.Second, time.Millisecond)
	assert.Equal(t, tea.QuitMsg{}, sentMsgs()[1])
	assert.Equal(t, tea.QuitMsg{}, g.filter(nil, tea.QuitMsg{}), "the resent quit goes through")
}

func TestReplyGuardQuitsWhenTerminalDoesNotAnswer(t *testing.T) {
	// sent stands in for the program's event loop, the only caller of filter.
	sent := make(chan tea.Msg, 4)
	g := &replyGuard{wait: time.Millisecond, send: func(m tea.Msg) { sent <- m }}
	assert.Nil(t, g.filter(nil, tea.InterruptMsg{}))

	// The cursor position request and the timeout race each other.
	got := []tea.Msg{<-sent, <-sent}
	assert.ElementsMatch(t, []tea.Msg{tea.RequestCursorPosition(), replyGuardTimeout{}}, got)

	assert.Nil(t, g.filter(nil, replyGuardTimeout{}), "the timeout is consumed")
	assert.Equal(t, tea.InterruptMsg{}, <-sent, "the held interrupt is resent")
}

type quitOnKeyModel struct{}

func (quitOnKeyModel) Init() tea.Cmd { return nil }

func (m quitOnKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}

func (quitOnKeyModel) View() tea.View { return tea.NewView("x") }

// fakeTerminal answers the queries a program writes to it, in order, starting
// only after delay, like a terminal at the end of a slow link.
type fakeTerminal struct {
	mu      sync.Mutex
	seen    string
	queries int
	replies chan string
	wrote   chan struct{}
}

var fakeTerminalReplies = []struct{ query, reply string }{
	{"\x1b[?2026$p", "\x1b[?2026;2$y"},
	{"\x1b[?2027$p", "\x1b[?2027;2$y"},
	{"\x1b[?u", "\x1b[?0u"},
	{"\x1b[6n", "\x1b[1;1R"},
}

func newFakeTerminal(in *os.File, delay time.Duration) *fakeTerminal {
	f := &fakeTerminal{replies: make(chan string, 16), wrote: make(chan struct{}, 16)}
	go func() {
		time.Sleep(delay)
		for r := range f.replies {
			_, _ = in.WriteString(r)
			f.wrote <- struct{}{}
		}
	}()
	return f
}

// Write answers queries in the order the program wrote them, including one
// split across two writes.
func (f *fakeTerminal) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen += string(b)
	for {
		reply, end := "", -1
		for _, q := range fakeTerminalReplies {
			if i := strings.Index(f.seen, q.query); i >= 0 && (end < 0 || i+len(q.query) < end) {
				reply, end = q.reply, i+len(q.query)
			}
		}
		if end < 0 {
			return len(b), nil
		}
		f.seen = f.seen[end:]
		f.queries++
		f.replies <- reply
	}
}

func (f *fakeTerminal) queriesSeen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queries
}

// runQuitImmediately runs a program that quits on the first key, typed right
// away, against a terminal that answers queries late. build makes the program
// from the options. It returns what was left unread on the program's input.
func runQuitImmediately(t *testing.T, build func(in *os.File, opts ...tea.ProgramOption) *tea.Program) string {
	inR, inW, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { inR.Close(); inW.Close() })
	term := newFakeTerminal(inW, 100*time.Millisecond)

	opts := []tea.ProgramOption{
		tea.WithInput(inR),
		tea.WithOutput(term),
		tea.WithEnvironment([]string{"TERM=xterm-256color"}),
		tea.WithWindowSize(80, 24),
		tea.WithoutSignalHandler(),
	}
	p := build(inR, opts...)
	go func() { _, _ = inW.WriteString("q") }()
	_, err = p.Run()
	require.NoError(t, err)

	// Wait for the terminal to have answered every query it saw.
	queries := term.queriesSeen()
	require.Positive(t, queries, "the program sent no queries")
	for range queries {
		select {
		case <-term.wrote:
		case <-time.After(5 * time.Second):
			t.Fatal("fake terminal did not answer")
		}
	}

	// Bubble Tea puts the pipe in blocking mode, so read deadlines don't
	// work: read up to a marker written after everything else instead.
	_, err = inW.WriteString("\x00")
	require.NoError(t, err)
	var left []byte
	buf := make([]byte, 64)
	for !bytes.HasSuffix(left, []byte{0}) {
		n, err := inR.Read(buf)
		require.NoError(t, err)
		left = append(left, buf[:n]...)
	}
	return string(bytes.TrimSuffix(left, []byte{0}))
}

func TestGuardedProgramConsumesLateReplies(t *testing.T) {
	// Without the guard the late answers are left for whoever reads next,
	// which is the user's shell.
	// Whether the kitty query went out depends on whether the first frame
	// was rendered before quitting.
	unguarded := func(in *os.File, opts ...tea.ProgramOption) *tea.Program {
		return tea.NewProgram(quitOnKeyModel{}, opts...)
	}
	assert.Contains(t, runQuitImmediately(t, unguarded), "\x1b[?2026;2$y\x1b[?2027;2$y")
	guarded := func(_ *os.File, opts ...tea.ProgramOption) *tea.Program {
		return newGuardedProgram(quitOnKeyModel{}, opts...)
	}
	assert.Empty(t, runQuitImmediately(t, guarded))
}

// TestNewTeaProgramGuardsTerminalInput runs the constructor the prompts use,
// with stdin posing as a terminal, so NewTeaProgram installs the guard. The
// quit waits for the terminal's cursor report, which arrives about 100 ms in,
// well inside replyWait, so the bound is not what ends the wait.
func TestNewTeaProgramGuardsTerminalInput(t *testing.T) {
	start := time.Now()
	left := runQuitImmediately(t, func(in *os.File, opts ...tea.ProgramOption) *tea.Program {
		return NewTeaProgram(quitOnKeyModel{}, FakeTTYReader(in), opts...)
	})
	assert.Empty(t, left)
	assert.Less(t, time.Since(start), replyWait)
}
