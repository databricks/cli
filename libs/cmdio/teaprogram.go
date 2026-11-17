package cmdio

import (
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

// replyWait bounds how long a quitting program waits for the terminal to
// answer. The wait is normally one round trip: milliseconds locally, longer
// over SSH. Practically every terminal answers a cursor position request, so
// the bound only applies to one that doesn't; it is long enough to cover a
// slow link, where a short bound would let the answers leak after all.
const replyWait = 2 * time.Second

// NewTeaProgram returns a Bubble Tea program for a TUI that reads keystrokes
// from in (os.Stdin when nil). When in is a terminal, quitting is deferred
// until the terminal has answered the queries Bubble Tea sent at startup.
//
// Bubble Tea v2 queries terminals that read input for synchronized output and
// unicode support (DECRQM 2026 and 2027, tea.go:1141) and for kitty keyboard
// flags (cursed_renderer.go:447). It doesn't wait for the answers, so when a
// program quits first (Enter or Ctrl+C right away, or a slow SSH link) they
// arrive after the terminal is handed back and land at the shell prompt as
// text like ^[[?2026;2$y.
func NewTeaProgram(m tea.Model, in io.Reader, opts ...tea.ProgramOption) *tea.Program {
	if in == nil {
		in = os.Stdin
	}
	if !isTTY(in) {
		return tea.NewProgram(m, opts...)
	}
	return newGuardedProgram(m, opts...)
}

func newGuardedProgram(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
	g := &replyGuard{wait: replyWait}
	p := tea.NewProgram(m, append(opts, tea.WithFilter(g.filter))...)
	g.send = p.Send
	return p
}

// replyGuard defers a program's quit until the terminal has answered a
// cursor position request sent at quit time. Terminals answer in order, so
// that answer arriving means every earlier query has been answered and read
// by the program too. Every terminal answers a cursor position request
// (CPR), unlike the startup queries themselves.
//
// Keystrokes typed while waiting are discarded: the model has already quit.
type replyGuard struct {
	wait time.Duration
	send func(tea.Msg)

	// Only the program's event loop calls filter, so these need no lock.
	waiting  bool
	released bool
	quit     tea.Msg
}

// replyGuardTimeout ends the wait when the terminal doesn't answer.
type replyGuardTimeout struct{}

func (g *replyGuard) filter(_ tea.Model, msg tea.Msg) tea.Msg {
	switch msg.(type) {
	case tea.QuitMsg, tea.InterruptMsg:
		if g.released {
			return msg
		}
		if !g.waiting {
			g.waiting = true
			g.quit = msg
			go g.send(tea.RequestCursorPosition())
			time.AfterFunc(g.wait, func() { g.send(replyGuardTimeout{}) })
		}
		return nil
	case tea.CursorPositionMsg, replyGuardTimeout:
		if g.waiting && !g.released {
			g.released = true
			go g.send(g.quit)
			return nil
		}
		if _, ok := msg.(replyGuardTimeout); ok {
			return nil
		}
	case tea.KeyPressMsg, tea.KeyReleaseMsg, tea.PasteMsg:
		if g.waiting {
			return nil
		}
	}
	return msg
}

// teaColorProfile returns the color profile for a program writing to w whose
// view mixes cmdio's template color helpers (raw 16-color SGR, gated on
// whether w supports color, passed as color) with lipgloss styles already
// rendered through a Renderer.
//
// Bubble Tea v1 passed both through as written. v2 downsamples the whole view
// to the program's profile, so the profile must not drop what v1 showed:
// basic ANSI at least when w supports color, so helper colors survive even
// where lipgloss styles render without color (e.g. TERM unset). Without color
// it is Ascii, not NoTTY, so bold, faint, italic and underline still apply.
func teaColorProfile(color bool, w io.Writer) colorprofile.Profile {
	if !color {
		return colorprofile.Ascii
	}
	return max(colorDepth(w), colorprofile.ANSI)
}
