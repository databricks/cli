// Package teainput decodes Bubble Tea input messages into one shape for all
// TUI models in the CLI.
//
// The shape follows Bubble Tea v2: keys are matched by their keystroke name
// ("enter", "ctrl+c", "space", "alt+backspace", "a") and printable input is
// carried as text, with bracketed paste as a separate event. Models written
// against [Event] don't depend on the version-specific key API, so moving to
// v2 only changes this package.
package teainput

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Event is a decoded key press or paste.
type Event struct {
	// Key is the keystroke name, e.g. "enter", "ctrl+c", "space",
	// "alt+backspace" or "a". It is empty when the message carries only text,
	// which is a paste, or several runes that arrived in one read; and for
	// several Alt-modified runes in one read, which carry no text either.
	Key string

	// Text is the printable text the event inserts. It is empty for
	// non-printable keys and for Alt-modified runes.
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
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return Event{}, false
	}
	if k.Paste {
		return Event{Text: string(k.Runes), Paste: true}, true
	}

	switch k.Type { //nolint:exhaustive // only runes and space carry Alt or text
	case tea.KeyRunes:
		if k.Alt {
			if len(k.Runes) == 1 {
				return Event{Key: "alt+" + string(k.Runes)}, true
			}
			return Event{}, true
		}
		// One rune is a key press. Several runes in one read are typed text
		// (or a paste without brackets) and get no key name, so a burst that
		// spells a key name, like "enter", is text and not that key.
		if len(k.Runes) == 1 {
			return printable(string(k.Runes)), true
		}
		return Event{Text: string(k.Runes)}, true
	case tea.KeySpace:
		if k.Alt {
			return Event{Key: "alt+space"}, true
		}
		return printable(" "), true
	}
	return Event{Key: k.String()}, true
}

// printable is the event for one typed rune: its key name is the rune, or
// "space" for a space.
func printable(s string) Event {
	if s == " " {
		return Event{Key: "space", Text: s}
	}
	return Event{Key: s, Text: s}
}
