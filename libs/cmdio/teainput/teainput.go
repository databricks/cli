// Package teainput decodes Bubble Tea input messages into one shape for all
// TUI models in the CLI.
//
// The shape follows Bubble Tea v2: keys are matched by their keystroke name
// ("enter", "ctrl+c", "space", "alt+backspace", "a") and printable input is
// carried as text, with bracketed paste as a separate event. Models written
// against [Event] don't depend on the version-specific key API, so a future
// Bubble Tea change only touches this package.
package teainput

import (
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// Event is a decoded key press or paste.
type Event struct {
	// Key is the keystroke name, e.g. "enter", "ctrl+c", "space",
	// "alt+backspace" or "a". It is empty when the message carries only text,
	// which is a paste or several runes that arrived in one read.
	Key string

	// Text is the printable text the event inserts. It is empty for
	// non-printable keys and for Alt- or Ctrl-modified keys.
	Text string

	// Paste is set when Text came from a bracketed paste.
	Paste bool
}

// Decode returns the [Event] for a key press or paste message, and false for
// every other message.
//
// Alt stays in the name, so Alt+Enter is "alt+enter" and no model matches it
// unless it names it. Alt+rune and Alt+space carry no text.
func Decode(msg tea.Msg) (Event, bool) {
	switch m := msg.(type) {
	case tea.PasteMsg:
		return Event{Text: m.Content, Paste: true}, true
	case tea.KeyPressMsg:
		return decodeKey(m), true
	}
	return Event{}, false
}

func decodeKey(m tea.KeyPressMsg) Event {
	if m.Mod.Contains(tea.ModAlt) || m.Mod.Contains(tea.ModCtrl) {
		return Event{Key: KeyName(m)}
	}
	if m.Text == "" {
		return Event{Key: KeyName(m)}
	}
	// One rune is a key press. Several runes in one read are typed text (or a
	// paste without brackets) and get no key name, so a burst that spells a key
	// name, like "enter", is text and not that key.
	if utf8.RuneCountInString(m.Text) > 1 {
		return Event{Text: m.Text}
	}
	if m.Text == " " {
		return Event{Key: "space", Text: m.Text}
	}
	return Event{Key: m.Text, Text: m.Text}
}

// KeyName returns the keystroke name models match on, for models that switch
// on [tea.KeyPressMsg] directly instead of going through [Decode].
//
// Shift+Enter is reported as "enter". Bubble Tea v2 enables the kitty keyboard
// protocol and modifyOtherKeys, so terminals that support them (Ghostty,
// kitty, WezTerm) send Shift+Enter as a distinct key; without those protocols
// it arrives as a plain CR, which is how users of these prompts know it.
func KeyName(m tea.KeyPressMsg) string {
	name := m.String()
	if m.Mod.Contains(tea.ModAlt) || m.Mod.Contains(tea.ModCtrl) {
		name = m.Keystroke()
	}
	if name == "shift+enter" {
		return "enter"
	}
	return name
}
