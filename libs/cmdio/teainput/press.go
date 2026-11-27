package teainput

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// v1KeyTypes maps the keystroke names used by [Event.Key] to v1 key types.
var v1KeyTypes = func() map[string]tea.KeyType {
	m := map[string]tea.KeyType{}
	// v1 key types are control codes (0..127) and negative special keys.
	for k := tea.KeyType(-128); k < 128; k++ {
		if name := k.String(); name != "" && name != " " {
			m[name] = k
		}
	}
	m["space"] = tea.KeySpace
	return m
}()

// Press builds the message for a key press, named like [Event.Key]
// (e.g. "enter", "ctrl+c", "alt+backspace", "space", "q"). It exists so model
// tests don't construct version-specific key messages. It panics on a name
// that is neither a known key nor a single rune, so a misspelled key fails
// the test instead of being typed as text.
func Press(key string) tea.Msg {
	alt := false
	if rest, ok := strings.CutPrefix(key, "alt+"); ok && rest != "" {
		alt, key = true, rest
	}
	if t, ok := v1KeyTypes[key]; ok {
		msg := tea.KeyMsg{Type: t, Alt: alt}
		if t == tea.KeySpace {
			msg.Runes = []rune{' '}
		}
		return msg
	}
	if utf8.RuneCountInString(key) != 1 {
		panic(fmt.Sprintf("teainput.Press: unknown key %q", key))
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key), Alt: alt}
}

// Type builds the message for typing text as a single event, like a burst of
// fast typing.
func Type(text string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}
}

// Paste builds the message for a bracketed paste of text.
func Paste(text string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text), Paste: true}
}
